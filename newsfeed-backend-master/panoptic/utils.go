package panoptic

import (
	"fmt"

	"google.golang.org/protobuf/encoding/prototext"
	"gorm.io/gorm"

	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/protocol"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

var MergebleSubSourceAllowList = []string{"推特", "微博", "雪球", "智堡"}

// Get sources that needs DB to add more subsources
// Returns a set of source ids
func GetCustomizedSubsourceSourceId(db *gorm.DB) map[string]bool {
	var sources []model.Source
	db.Where("name IN ?", MergebleSubSourceAllowList).Find(&sources)
	sourceIdsToReadSubsourceFromDB := make(map[string]bool)
	for _, source := range sources {
		sourceIdsToReadSubsourceFromDB[source.Id] = true
	}
	return sourceIdsToReadSubsourceFromDB
}

// Add more sub sources from DB.
// This function updates configs, returns nothing
func MergeSubsourcesFromConfigAndDb(db *gorm.DB, configs *protocol.PanopticConfigs) {
	sourceIdsWithSubsourceFromDB := GetCustomizedSubsourceSourceId(db)
	// For all sources
	for _, config := range configs.Config {
		isCustomizedCrawler := (config.DataCollectorId == protocol.PanopticTask_COLLECTOR_USER_CUSTOMIZED_SUBSOURCE) || (config.DataCollectorId == protocol.PanopticTask_COLLECTOR_USER_CUSTOMIZED_SOURCE)
		// Add subsources only for Weibo and the one support customized subsource by user
		if _, ok := sourceIdsWithSubsourceFromDB[config.TaskParams.SourceId]; !ok && !isCustomizedCrawler {
			continue
		}
		param := config.TaskParams
		var subSourcesFromDB []model.SubSource

		if isCustomizedCrawler {
			db.Where("source_id = ? AND is_from_shared_post = false AND customized_crawler_params IS NOT NULL", param.SourceId).Order("name").Find(&subSourcesFromDB)
		} else {
			db.Where("source_id = ? AND is_from_shared_post = false AND customized_crawler_params IS NULL", param.SourceId).Order("name").Find(&subSourcesFromDB)
		}

		existingSubSourceMap := map[string]bool{}
		// subsource name is unique, using it to do lookup
		for _, s := range param.SubSources {
			existingSubSourceMap[s.Name] = true
		}

		for _, s := range subSourcesFromDB {
			// only use subsources from DB that is not in config
			if _, ok := existingSubSourceMap[s.Name]; !ok {
				// This ptr is holding crawler params for a subsource, can be null
				var crawlerParamsPtr *protocol.CustomizedCrawlerParams

				if s.CustomizedCrawlerParams != nil {
					// This obj is to receive unmarshal result, if succeeded, crawlerParamsPtr will point to it
					var panopticConfig protocol.CustomizedCrawlerParams
					if err := prototext.Unmarshal([]byte(*s.CustomizedCrawlerParams), &panopticConfig); err != nil {
						Logger.LogV2.Error(fmt.Sprintf("can't unmarshal customized crawler param for subsource %s, error %+v", s.Name, err))
						continue
					}
					crawlerParamsPtr = &panopticConfig
				}
				param.SubSources = append(param.SubSources, &protocol.PanopticSubSource{
					Name:                                s.Name,
					Type:                                protocol.PanopticSubSource_USERS, // default to users type
					ExternalId:                          s.ExternalIdentifier,
					Link:                                s.OriginUrl,
					AvatarUrl:                           &s.AvatarUrl,
					CustomizedCrawlerParamsForSubSource: crawlerParamsPtr,
				})
			}
		}
	}
}
