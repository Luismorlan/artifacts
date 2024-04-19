package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	"github.com/rnr-capital/newsfeed-backend/server/resolver"
	"github.com/rnr-capital/newsfeed-backend/utils"
	"github.com/rnr-capital/newsfeed-backend/utils/dotenv"
	"google.golang.org/protobuf/encoding/prototext"
)

const (
	DataDir = "collector/cmd/data"
	JobId   = "wublock123_job"
)

// Run this to initialize subsources for wublock123
// NEWSMUX_ENV=prod go run scripts/add_wublock123_subsources/main.go

// Index all panoptic jobs in data folder, by the job id
// Copied from collector/cmd/main.go
func ParseAndIndexPanopticJobs() map[string]*protocol.PanopticJob {
	files, err := os.ReadDir(DataDir)
	if err != nil {
		log.Fatalln(err)
	}

	res := []byte{}
	for _, file := range files {
		in, err := os.ReadFile(filepath.Join(DataDir, file.Name()))
		if err != nil {
			log.Fatalln(err)
		}
		res = append(res, in...)
	}

	jobs := &protocol.PanopticJobs{}
	if err := prototext.Unmarshal(res, jobs); err != nil {
		log.Fatalln(err)
	}

	index := make(map[string]*protocol.PanopticJob)
	for _, job := range jobs.Jobs {
		if _, ok := index[job.JobId]; ok {
			log.Fatalln("duplicate job id in testing directory: ", job.JobId)
		}
		index[job.JobId] = job
	}

	return index
}

func main() {
	if err := dotenv.LoadDotEnvs(); err != nil {
		panic(err)
	}

	db, err := utils.GetDBConnection()
	if err != nil {
		log.Fatalln(err)
	}

	index := ParseAndIndexPanopticJobs()
	job, ok := index[JobId]
	if !ok {
		log.Fatalln("Wublock123 job not found")
	}

	ctx := context.Background()

	for _, channel := range job.Tasks[0].TaskParams.GetWublock123TaskParams().Channels {
		fmt.Printf("Adding channel %s\n", channel)
		resolver.AddSubSourceImp(db, ctx, model.AddSubSourceInput{
			SourceID:          job.Tasks[0].TaskParams.SourceId,
			SubSourceUserName: channel,
		})
	}

	for _, keyword := range job.Tasks[0].TaskParams.GetWublock123TaskParams().SearchKeywords {
		fmt.Printf("Adding keyword %s\n", keyword)
		resolver.AddSubSourceImp(db, ctx, model.AddSubSourceInput{
			SourceID:          job.Tasks[0].TaskParams.SourceId,
			SubSourceUserName: keyword,
		})
	}

}
