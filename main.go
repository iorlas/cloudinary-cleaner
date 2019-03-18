package main

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

func main() {
	var err error
	viper.AutomaticEnv()

	viper.SetDefault("query", "uploaded_at<1d AND context.unused=true")
	viper.SetDefault("debug", true)
	viper.SetDefault("count", 100)
	viper.SetDefault("timeout", 10)

	var log *zap.Logger

	if viper.GetBool("debug") {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		panic(err)
	}

	cloudinary, err := NewCloudinary(viper.GetViper(), log)
	if err != nil {
		panic(err)
	}

	var query = viper.GetString("query")
	var count = viper.GetInt("count")
	var timeout = viper.GetDuration("timeout")

	for {
		var results, err = cloudinary.Search(query, count)
		if err != nil {
			log.Error(err.Error())

			time.Sleep(timeout * time.Second)
			continue
		}

		if results.TotalCount > 0 {
			var ids []string
			for _, res := range results.Resources {
				ids = append(ids, res.PublicID)
			}

			log.Info("Found objects to delete",
				zap.Int("count", len(results.Resources)),
				zap.Strings("ids", ids))

			var _, err = cloudinary.BatchDelete(ids)
			if err != nil {
				log.Error(err.Error())

				time.Sleep(timeout * time.Second)
				continue
			}

			log.Info("Objects deleted",
				zap.Int("count", len(results.Resources)),
				zap.Strings("ids", ids))
		}

		time.Sleep(timeout * time.Second)
	}
}
