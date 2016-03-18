package ec2cluster

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// LifecycleMessage represents the message we receive from the
// autoscaling lifecycle hook when an instance is created or
// terminated.
type LifecycleMessage struct {
	AutoScalingGroupName string    `json:",omitempty"`
	Service              string    `json:",omitempty"`
	Time                 time.Time `json:",omitempty"`
	AccountID            string    `json:",omitempty"`
	LifecycleTransition  string    `json:",omitempty"`
	RequestID            string    `json:"RequestId"`
	LifecycleActionToken string    `json:",omitempty"`
	EC2InstanceID        string    `json:"EC2InstanceID"`
	LifecycleHookName    string    `json:",omitempty"`
}

// LifecyleEventCallback is a function that is invoked for each
// ASG lifecycle event. If the function returns a non-nil error
// then the message remains in the queue. If `shouldContinue` is
// true then CompleteLifecycleAction() is invoked with `CONINTUE`
// otherwise it is invoked with `ABANDON`.
type LifecyleEventCallback func(m *LifecycleMessage) (shouldContinue bool, err error)

// WatchLifecycleEvents monitors a lifecycle event SQS queue and invokes
// cb for each event. The current instance's autoscaling group is
// inspected to determine the URL of the queue to use.
func (s *Cluster) WatchLifecycleEvents(cb LifecyleEventCallback) error {
	sqsSvc := sqs.New(s.AwsSession)
	autoscalingSvc := autoscaling.New(s.AwsSession)

	queueURL := ""
	{
		asg, err := s.AutoscalingGroup()
		if err != nil {
			return err
		}
		resp, err := autoscalingSvc.DescribeLifecycleHooks(&autoscaling.DescribeLifecycleHooksInput{
			AutoScalingGroupName: asg.AutoScalingGroupName,
		})
		if err != nil {
			return err
		}
		for _, hook := range resp.LifecycleHooks {
			if !strings.HasPrefix(*hook.NotificationTargetARN, "arn:aws:sqs:") {
				continue
			}
			arnParts := strings.Split(*hook.NotificationTargetARN, ":")
			queueName := arnParts[len(arnParts)-1]
			queueOwnerAWSAccountID := arnParts[len(arnParts)-2]

			resp, err := sqsSvc.GetQueueUrl(&sqs.GetQueueUrlInput{
				QueueName:              &queueName,
				QueueOwnerAWSAccountId: &queueOwnerAWSAccountID,
			})
			if err != nil {
				return err
			}
			queueURL = *resp.QueueUrl
			break
		}
		if queueURL == "" {
			return fmt.Errorf("cannot find any lifecycle hooks for %s",
				*asg.AutoScalingGroupName)
		}
	}

	for {
		resp, err := sqsSvc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            &queueURL,
			MaxNumberOfMessages: aws.Int64(1),
			WaitTimeSeconds:     aws.Int64(20),
		})
		if err != nil {
			return err
		}
		for _, messageWrapper := range resp.Messages {
			m := LifecycleMessage{}
			if err := json.Unmarshal([]byte(*messageWrapper.Body), &m); err != nil {
				return fmt.Errorf("cannot unmarshal event: %s", err)
			}
			if m.LifecycleTransition != "autoscaling:EC2_INSTANCE_LAUNCHING" && m.LifecycleTransition != "autoscaling:EC2_INSTANCE_TERMINATING" {
				sqsSvc.DeleteMessage(&sqs.DeleteMessageInput{
					QueueUrl:      &queueURL,
					ReceiptHandle: messageWrapper.ReceiptHandle,
				})
				continue
			}

			shouldContinue, err := cb(&m)
			if err != nil {
				continue
			}
			lifecycleActionResult := "CONTINUE"
			if !shouldContinue {
				lifecycleActionResult = "ABANDON"
			}

			_, err = autoscalingSvc.CompleteLifecycleAction(&autoscaling.CompleteLifecycleActionInput{
				AutoScalingGroupName:  &m.AutoScalingGroupName,
				LifecycleActionResult: aws.String(lifecycleActionResult),
				LifecycleHookName:     &m.LifecycleHookName,
				InstanceId:            &m.EC2InstanceID,
				LifecycleActionToken:  &m.LifecycleActionToken,
			})
			if err != nil {
				return err
			}

			_, err = sqsSvc.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      &queueURL,
				ReceiptHandle: messageWrapper.ReceiptHandle,
			})
			if err != nil {
				return err
			}
		}
	}
}
