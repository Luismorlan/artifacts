package modules

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/rnr-capital/newsfeed-backend/panoptic"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
	"google.golang.org/protobuf/proto"
)

type ReporterConfig struct {
	Name string
}

// Reporter's job is to listen to different channels and aggregate results,
// sending to Datadog (Or other service if there's any) for monitoring purpose.
type Reporter struct {
	panoptic.Module

	Config ReporterConfig

	Statsd *statsd.Client

	EventBus *gochannel.GoChannel
}

func NewReporter(config ReporterConfig, statsd *statsd.Client, e *gochannel.GoChannel) *Reporter {
	return &Reporter{
		Config:   config,
		Statsd:   statsd,
		EventBus: e,
	}
}

// Report task result state to datadog. Each finished task increment the task
// counter by 1, and tag it with lots of other information in order for backend
// to slice it.
func ReportTaskResultState(task *protocol.PanopticTask, statsdClient *statsd.Client) {
	err := statsdClient.Incr(panoptic.DdogTaskStateCounter,
		[]string{
			task.TaskMetadata.ConfigName,
			task.DataCollectorId.String(),
			task.TaskMetadata.IpAddr,
			task.TaskMetadata.ResultState.String(),
		}, 1)
	if err != nil {
		Logger.LogV2.Info("cannot report result state")
	}
}

// Report how many messages are crawled or failed for a task.
func ReportTaskMessages(task *protocol.PanopticTask, statsdClient *statsd.Client) {
	err := statsdClient.Count(panoptic.DdogTaskSuccessMessageCounter,
		int64(task.TaskMetadata.TotalMessageCollected),
		[]string{
			task.TaskMetadata.ConfigName,
			task.DataCollectorId.String(),
			task.TaskMetadata.IpAddr,
			task.TaskMetadata.ResultState.String(),
		}, 1)
	if err != nil {
		Logger.LogV2.Info("cannot report total message count")
	}

	err = statsdClient.Count(panoptic.DdogTaskFailureMessageCounter,
		int64(task.TaskMetadata.TotalMessageFailed),
		[]string{
			task.TaskMetadata.ConfigName,
			task.DataCollectorId.String(),
			task.TaskMetadata.IpAddr,
			task.TaskMetadata.ResultState.String(),
		}, 1)
	if err != nil {
		Logger.LogV2.Info("cannot report total message failed")
	}
}

// Report how many seconds the given task took to execute.
func ReportTaskExecutionTime(task *protocol.PanopticTask, statsdClient *statsd.Client) {
	// print out task
	fmt.Printf("ReportTaskExecutionTime task: %s", task.String())

	statsdClient.Distribution(panoptic.DdogTaskExecutionTimeDistribution,
		float64(task.TaskMetadata.TaskEndTime.Seconds-task.TaskMetadata.TaskStartTime.Seconds),
		[]string{
			task.TaskMetadata.ConfigName,
			task.DataCollectorId.String(),
			task.TaskMetadata.ResultState.String(),
		}, 1)
}

// Report task level tracking information.
func (r *Reporter) ReportTask(job *protocol.PanopticJob) {
	for _, task := range job.Tasks {
		ReportTaskResultState(task, r.Statsd)
		ReportTaskMessages(task, r.Statsd)
		ReportTaskExecutionTime(task, r.Statsd)
	}
}

func (r *Reporter) ProcessPanopticJobs(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messages, err := r.EventBus.Subscribe(ctx, panoptic.TopicExecutedJob)
	if err != nil {
		return err
	}

	for msg := range messages {
		msg.Ack()

		job := protocol.PanopticJob{}
		err := proto.Unmarshal(msg.Payload, &job)

		if err != nil {
			return err
		}

		Logger.LogV2.Info(fmt.Sprintf("reporter received PanopticJob: %s", job.String()))

		// Export metrics to Datadog only if we're in prod environment, so that
		// local testing won't pollute the Datadog dashboard.
		if !utils.IsProdEnv() {
			continue
		}
		r.ReportTask(&job)
	}

	return nil
}

func (r *Reporter) RunModule(ctx context.Context) error {
	r.ProcessPanopticJobs(ctx)
	return nil
}

func (r *Reporter) Name() string {
	return r.Config.Name
}

func (r *Reporter) Shutdown() {
	Logger.LogV2.Info(fmt.Sprint("Module ", r.Config.Name, " gracefully shutdown"))
}
