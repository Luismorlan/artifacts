package main

import (
	"context"
	"fmt"

	"github.com/rnr-capital/newsfeed-backend/app_setting"
	"github.com/rnr-capital/newsfeed-backend/collector"
	"github.com/rnr-capital/newsfeed-backend/panoptic/modules"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
)

func main() {
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	s := modules.NewScheduler(
		&app_setting.PanopticAppSetting{
			FORCE_REMOTE_SCHEDULE_PULL: true,
		},
		modules.SchedulerConfig{},
		nil,
		&modules.PrinterJobDoer{},
		context.Background())
	configs, _, err := s.ReadConfig()
	if err != nil {
		panic(err)
	}

	fmt.Println(collector.PrettyPrint(configs))
}
