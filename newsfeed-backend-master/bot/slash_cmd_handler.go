package bot

// This handler is for slack slash commands
// https://api.slack.com/interactivity/slash-commands

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/slack-go/slack"
	"gorm.io/gorm"

	"github.com/rnr-capital/newsfeed-backend/bot/articlesaver"
	"github.com/rnr-capital/newsfeed-backend/model"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

const (
	SubscribeButtonText   = "Subscribe"
	UnsubscribeButtonText = "Unsubscribe"
	Endings               = `
   {
      "indexId": "6827182718212",
      "configType": 2,
      "configVersion": 2,
      "sort": 10,
      "address": "panoramic 新加坡.conf",
      "port": 0,
      "id": "",
      "alterId": 0,
      "security": "",
      "network": "",
      "remarks": "panoramic 新加坡 0.5",
      "headerType": "",
      "requestHost": "",
      "path": "",
      "streamSecurity": "",
      "allowInsecure": "False",
      "testResult": "",
      "subid": "",
      "flow": "",
      "sni": null,
      "alpn": null,
      "groupId": "",
      "coreType": 22,
      "preSocksPort": 0
    },
   {
      "indexId": "4924282718212",
      "configType": 2,
      "configVersion": 2,
      "sort": 10,
      "address": "panoramic 日本.conf",
      "port": 0,
      "id": "",
      "alterId": 0,
      "security": "",
      "network": "",
      "remarks": "panoramic 日本 0.9",
      "headerType": "",
      "requestHost": "",
      "path": "",
      "streamSecurity": "",
      "allowInsecure": "False",
      "testResult": "",
      "subid": "",
      "flow": "",
      "sni": null,
      "alpn": null,
      "groupId": "",
      "coreType": 22,
      "preSocksPort": 0
    },
   {
      "indexId": "78271872718212",
      "configType": 2,
      "configVersion": 2,
      "sort": 10,
      "address": "panoramic 东京.conf",
      "port": 0,
      "id": "",
      "alterId": 0,
      "security": "",
      "network": "",
      "remarks": "panoramic 东京 0.5",
      "headerType": "",
      "requestHost": "",
      "path": "",
      "streamSecurity": "",
      "allowInsecure": "False",
      "testResult": "",
      "subid": "",
      "flow": "",
      "sni": null,
      "alpn": null,
      "groupId": "",
      "coreType": 22,
      "preSocksPort": 0
    },
   {
      "indexId": "5827182718212",
      "configType": 2,
      "configVersion": 2,
      "sort": 10,
      "address": "panoramic 洛杉矶.conf",
      "port": 0,
      "id": "",
      "alterId": 0,
      "security": "",
      "network": "",
      "remarks": "panoramic 洛杉矶 0.5",
      "headerType": "",
      "requestHost": "",
      "path": "",
      "streamSecurity": "",
      "allowInsecure": "False",
      "testResult": "",
      "subid": "",
      "flow": "",
      "sni": null,
      "alpn": null,
      "groupId": "",
      "coreType": 22,
      "preSocksPort": 0
    },
    {
      "indexId": "4827182719212",
      "configType": 2,
      "configVersion": 2,
      "sort": 10,
      "address": "panoramic 香港.conf",
      "port": 0,
      "id": "",
      "alterId": 0,
      "security": "",
      "network": "",
      "remarks": "panoramic 香港 0.5",
      "headerType": "",
      "requestHost": "",
      "path": "",
      "streamSecurity": "",
      "allowInsecure": "False",
      "testResult": "",
      "subid": "",
      "flow": "",
      "sni": null,
      "alpn": null,
      "groupId": "",
      "coreType": 22,
      "preSocksPort": 0
    }
	],
  "kcpItem": {
    "mtu": 1350,
    "tti": 50,
    "uplinkCapacity": 12,
    "downlinkCapacity": 100,
    "congestion": false,
    "readBufferSize": 2,
    "writeBufferSize": 2
  },
  "subItem": [
    {
      "id": "",
      "remarks": "remarks",
      "url": "url",
      "enabled": true,
      "userAgent": "",
      "groupId": ""
    }
  ],
  "uiItem": {
    "enableAutoAdjustMainLvColWidth": true,
    "mainLocation": "715, 349",
    "mainSize": "2061, 2015",
    "mainLvColWidth": {
      "def": 30,
      "configType": 288,
      "remarks": 277,
      "address": 120,
      "port": 100,
      "security": 120,
      "network": 120,
      "streamSecurity": 100,
      "subRemarks": 100,
      "testResult": 120
    }
  },
  "routings": [
    {
      "remarks": "绕过大陆(Whitelist)",
      "url": "",
      "rules": [
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "direct",
          "ip": null,
          "domain": [
            "domain:example-example.com",
            "domain:example-example2.com"
          ],
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "block",
          "ip": null,
          "domain": [
            "geosite:category-ads-all"
          ],
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "direct",
          "ip": null,
          "domain": [
            "geosite:cn"
          ],
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "direct",
          "ip": [
            "geoip:private",
            "geoip:cn"
          ],
          "domain": null,
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": "0-65535",
          "inboundTag": null,
          "outboundTag": "proxy",
          "ip": null,
          "domain": null,
          "protocol": null,
          "enabled": true
        }
      ],
      "enabled": true,
      "locked": false,
      "customIcon": null
    },
    {
      "remarks": "黑名单(Blacklist)",
      "url": "",
      "rules": [
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "direct",
          "ip": null,
          "domain": null,
          "protocol": [
            "bittorrent"
          ],
          "enabled": true
        },
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "block",
          "ip": null,
          "domain": [
            "geosite:category-ads-all"
          ],
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "proxy",
          "ip": [
            "geoip:telegram"
          ],
          "domain": [
            "geosite:gfw",
            "geosite:greatfire",
            "geosite:tld-!cn"
          ],
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": "0-65535",
          "inboundTag": null,
          "outboundTag": "direct",
          "ip": null,
          "domain": null,
          "protocol": null,
          "enabled": true
        }
      ],
      "enabled": true,
      "locked": false,
      "customIcon": null
    },
    {
      "remarks": "全局(Global)",
      "url": "",
      "rules": [
        {
          "type": null,
          "port": "0-65535",
          "inboundTag": null,
          "outboundTag": "proxy",
          "ip": null,
          "domain": null,
          "protocol": null,
          "enabled": true
        }
      ],
      "enabled": true,
      "locked": false,
      "customIcon": null
    },
    {
      "remarks": "locked",
      "url": "",
      "rules": [
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "proxy",
          "ip": null,
          "domain": [
            "geosite:google"
          ],
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "direct",
          "ip": null,
          "domain": [
            "domain:example-example.com",
            "domain:example-example2.com"
          ],
          "protocol": null,
          "enabled": true
        },
        {
          "type": null,
          "port": null,
          "inboundTag": null,
          "outboundTag": "block",
          "ip": null,
          "domain": [
            "geosite:category-ads-all"
          ],
          "protocol": null,
          "enabled": true
        }
      ],
      "enabled": true,
      "locked": true,
      "customIcon": null
    }
  ],
  "constItem": {
    "speedTestUrl": "http://cachefly.cachefly.net/10mb.test",
    "speedPingTestUrl": "https://www.google.com/generate_204",
    "defIEProxyExceptions": "localhost;127.*;10.*;172.16.*;172.17.*;172.18.*;172.19.*;172.20.*;172.21.*;172.22.*;172.23.*;172.24.*;172.25.*;172.26.*;172.27.*;172.28.*;172.29.*;172.30.*;172.31.*;192.168.*"
  },
  "globalHotkeys": null,
  "groupItem": [],
  "coreTypeItem": null
}`
)

func generateConfs(userInputs []string) {
	os.Mkdir("pano_ladder/guiConfigs", 0755)
	os.WriteFile("pano_ladder/guiConfigs/panoramic 香港.conf", []byte(`{
	"listen": "http://127.0.0.1:10809",
	"proxy": "https://pano:172912@cdnhk.rnr.capital:443"
}`), 0644)
	os.WriteFile("pano_ladder/guiConfigs/panoramic 东京.conf", []byte(`{
	"listen": "http://127.0.0.1:10809",
	"proxy": "https://pano:172912@cdnjp.rnr.capital:443"
}`), 0644)
	os.WriteFile("pano_ladder/guiConfigs/panoramic 日本.conf", []byte(`{
	"listen": "http://127.0.0.1:10809",
	"proxy": "https://pano:172912@cdnbwg.rnr.capital:443"
}`), 0644)
	os.WriteFile("pano_ladder/guiConfigs/panoramic 新加坡.conf", []byte(`{
	"listen": "http://127.0.0.1:10809",
	"proxy": "https://pano:172912@cdnsg.rnr.capital:443"
}`), 0644)
	os.WriteFile("pano_ladder/guiConfigs/panoramic 洛杉矶.conf", []byte(`{
	"listen": "http://127.0.0.1:10809",
	"proxy": "https://pano:172912@cdnla.rnr.capital:443"
}`), 0644)
	guiNConfig := `{
  "logEnabled": false,
  "loglevel": "warning",
  "indexId": "0",
  "muxEnabled": false,
  "sysProxyType": 0,
  "enableStatistics": false,
  "keepOlderDedupl": false,
  "statisticsFreshRate": 1,
  "remoteDNS": null,
  "domainStrategy4Freedom": null,
  "defAllowInsecure": false,
  "domainStrategy": "IPIfNonMatch",
  "domainMatcher": null,
  "routingIndex": 0,
  "enableRoutingAdvanced": true,
  "ignoreGeoUpdateCore": false,
  "systemProxyExceptions": null,
  "systemProxyAdvancedProtocol": null,
  "autoUpdateInterval": 0,
  "autoUpdateSubInterval": 0,
  "checkPreReleaseUpdate": false,
  "enableSecurityProtocolTls13": false,
  "trayMenuServersLimit": 30,
  "inbound": [
    {
      "localPort": 10808,
      "protocol": "socks",
      "udpEnabled": true,
      "sniffingEnabled": true,
      "allowLANConn": false,
      "user": null,
      "pass": null
    }
  ],
  "vmess": [`
	for i, userInput := range userInputs {
		segs := strings.Split(userInput, ",")
		if len(segs) != 5 {
			Logger.LogV2.Warn("Invalid user input" + userInput)
			continue
		}
		seg1 := segs[0]
		seg1s := strings.Split(seg1, " = ")
		if len(seg1s) != 2 {
			Logger.LogV2.Warn("Invalid user input" + userInput)
			continue
		}

		var re = regexp.MustCompile(`:.+:`)
		confname := re.ReplaceAllString(seg1s[0], "")
		f, err := os.Create("pano_ladder/guiConfigs/" + confname + ".conf")
		if err != nil {
			Logger.LogV2.Error("Can't create config file" + err.Error())
		}
		Logger.LogV2.Error(fmt.Sprintf("Failed to create config file %s", err))
		f.Write([]byte(fmt.Sprintf(`{
	"listen": "http://127.0.0.1:10809",
	"proxy": "%s://%s:%s@%s:%s"
}`, strings.TrimSpace(seg1s[1]), strings.TrimSpace(segs[3]), strings.TrimSpace(segs[4]), strings.TrimSpace(segs[1]), strings.TrimSpace(segs[2]))))
		f.Close()
		guiNConfig += fmt.Sprintf(`   {
      "indexId": "%d",
      "configType": 2,
      "configVersion": 2,
      "sort": 10,
      "address": "%s",
      "port": 0,
      "id": "",
      "alterId": 0,
      "security": "",
      "network": "",
      "remarks": "%s",
      "headerType": "",
      "requestHost": "",
      "path": "",
      "streamSecurity": "",
      "allowInsecure": "False",
      "testResult": "",
      "subid": "",
      "flow": "",
      "sni": null,
      "alpn": null,
      "groupId": "",
      "coreType": 22,
      "preSocksPort": 0
    },
`, i, confname+".conf", confname)
	}
	guiNConfig += Endings
	err := os.WriteFile("pano_ladder/guiNConfig.json", []byte(guiNConfig), 0644)
	if err != nil {
		Logger.LogV2.Error(fmt.Sprintf("Failed to write guiNConfig.json %s", err))
		fmt.Println("dude", err)
	}
}

func generateZip() (io.Reader, error) {
	b := &bytes.Buffer{}
	// 1. Create a ZIP file and zip.Writer
	writer := zip.NewWriter(b)
	defer writer.Close()

	source := "pano_ladder"
	// 2. Go through all the files of the source
	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
	return b, err
}

type CommandForm struct {
	Command     string `form:"command" binding:"required"`
	ChannelId   string `form:"channel_id" binding:"required"`
	UserId      string `form:"user_id" binding:"required"`
	ResponseUrl string `form:"response_url" binding:"required"`
}

func buildUserSubscribedColumnsMessageBody(columns []*model.Column) slack.Message {
	// subscribe section
	divSection := slack.NewDividerBlock()
	blocks := []slack.Block{divSection}

	sort.Slice(columns, func(i, j int) bool {
		return columns[i].Name < columns[j].Name
	})

	// columns this channel hasn't subscribed
	for _, column := range columns {
		if len(column.SubscribedChannels) == 0 {
			creator := ""
			if column.Creator.Name != "" {
				creator = fmt.Sprintf("_%s_", column.Creator.Name)
			}
			subscribeBtnText := slack.NewTextBlockObject("plain_text", SubscribeButtonText, false, false)
			subscribeBtnEle := slack.NewButtonBlockElement(column.Id, column.Name, subscribeBtnText)
			optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t %s", column.Name, creator), false, false)
			optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
			blocks = append(blocks, optionSection)
		}
	}

	blocks = append(blocks, divSection)
	// columns this channel has subscribed
	for _, column := range columns {
		if len(column.SubscribedChannels) == 1 {
			creator := ""
			if column.Creator.Name != "" {
				creator = fmt.Sprintf("_%s_", column.Creator.Name)
			}
			subscribeBtnText := slack.NewTextBlockObject("plain_text", UnsubscribeButtonText, false, false)
			subscribeBtnEle := slack.NewButtonBlockElement(column.Id, column.Name, subscribeBtnText)
			optionText := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* \t %s", column.Name, creator), false, false)
			optionSection := slack.NewSectionBlock(optionText, nil, slack.NewAccessory(subscribeBtnEle))
			blocks = append(blocks, optionSection)
		}
	}

	return slack.NewBlockMessage(blocks...)
}

func sanityCheck(text string) int {
	t := strings.TrimSpace(text)
	nodes := strings.Split(t, "\n")
	r := 0
	for i := 0; i < len(nodes); i++ {
		if len(strings.Split(strings.TrimSpace(nodes[i]), ",")) == 5 {
			r += 1
		}
	}
	return r
}

func SlashCommandHandler(db *gorm.DB, mu *sync.Mutex) gin.HandlerFunc {
	return func(c *gin.Context) {
		var form CommandForm
		c.Bind(&form)
		var newsCommand = "/news"
		if !utils.IsProdEnv() {
			newsCommand = "/devnews"
		}
		var vpnCommand = "/vpn"
		if !utils.IsProdEnv() {
			vpnCommand = "/devvpn"
		}
		var weiboCommand = "/weibo"
		if !utils.IsProdEnv() {
			weiboCommand = "/devweibo"
		}
		switch form.Command {
		case newsCommand:
			var channel model.Channel
			err := db.Model(&model.Channel{}).Where("channel_slack_id = ?", form.ChannelId).First(&channel).Error
			if err != nil {
				webhookMsg := &slack.WebhookMessage{
					Text: "The bot is not added to this channel yet, please add bot to this channel first: " + os.Getenv("BOT_ADDING_URL"),
				}
				slack.PostWebhook(form.ResponseUrl, webhookMsg)
				return
			}
			var user model.User
			if err := db.Model(&model.User{}).Preload("SubscribedColumns.Creator", "slack_id != ?", form.UserId).
				Preload("SubscribedColumns.SubscribedChannels", "channel_slack_id = ?", form.ChannelId).
				Where("slack_id = ?", form.UserId).
				First(&user).Error; err != nil {
				Logger.LogV2.Error(fmt.Sprint("failed to get user's columns", err))
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to get public columns. please contact tech"})
				return
			}

			msg := buildUserSubscribedColumnsMessageBody(user.SubscribedColumns)
			b, err := json.MarshalIndent(msg, "", "    ")
			if err != nil {
				Logger.LogV2.Error(fmt.Sprint("failed to build the message", err))
				return
			}
			c.Data(http.StatusOK, "application/json", b)
		case vpnCommand:
			trackingId := utils.RandomAlphabetString(10) + ": "
			Logger.LogV2.Info(trackingId + "got vpn command")
			s, err := slack.SlashCommandParse(c.Request)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to parse content"})
				return
			}
			nodes := sanityCheck(s.Text)
			if nodes == 0 {
				c.Data(http.StatusOK, "application/json", []byte("Uploading panoramic vpn client. It should take less than a minute"))
			} else {
				c.Data(http.StatusOK, "application/json", []byte(fmt.Sprintf(`Found %d blackssl nodes. Creating your vpn client, it usually takes a minutes.
Contact Boning if file isn't uploaded in 2 minutes. requestId: %s`, nodes, trackingId)))
			}

			api := slack.New(os.Getenv("BOT_TOKEN"))
			go func() {
				mu.Lock()
				generateConfs(strings.Split(strings.TrimSpace(s.Text), "\n"))
				f, err := generateZip()
				if err != nil {
					Logger.LogV2.Error(fmt.Sprint(trackingId, "failed to generate zip", err))
					return
				}
				params := slack.FileUploadParameters{
					Title:    "vpn client",
					Reader:   f,
					Filetype: "zip",
					Filename: "ladder.zip",
					Channels: []string{s.ChannelID},
				}
				qrparams := slack.FileUploadParameters{
					Title:    "Pano vpn shadowrocket qrcode",
					File:     "pano_ladder/qrcodes/qrcode.jpg",
					Channels: []string{s.ChannelID},
				}
				Logger.LogV2.Info(trackingId + "start to upload file")
				api.UploadFile(qrparams)
				file, err := api.UploadFile(params)
				Logger.LogV2.Info(trackingId + "file sent to user:" + file.ID)
				if err != nil {
					Logger.LogV2.Error(fmt.Sprint(trackingId, "failed to zip vpn client", err))
					return
				}
				os.RemoveAll("pano_ladder/guiConfigs/")
				os.Remove("pano_ladder/guiNConfig.json")
				defer mu.Unlock()
			}()
		case weiboCommand:
			s, err := slack.SlashCommandParse(c.Request)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"text": "failed to parse content"})
				return
			}
			// TODO: add some sanity check
			c.JSON(http.StatusOK, gin.H{
				"text":          "saving your article",
				"response_type": "ephemeral",
			})
			api := slack.New(os.Getenv("BOT_TOKEN"))
			go func(c *gin.Context) {
				url := s.Text
				doc, err := articlesaver.GetWeiboArticle(url)
				if err != nil {
					api.PostMessage(s.ChannelID, slack.MsgOptionText("failed to save content. Please contact tech team. "+err.Error(), false))
					return
				}
				link, err := articlesaver.SaveWeiboDocToNotion(doc, url)
				if err != nil {
					api.PostMessage(s.ChannelID, slack.MsgOptionText("failed to save content. Please contact tech team. "+err.Error(), false))
					return
				}
				api.PostMessage(s.ChannelID, slack.MsgOptionText(fmt.Sprintf("Article saved to notion: %s", link), false))
			}(c)
		default:
			c.JSON(http.StatusNotFound, gin.H{
				"response_type": "ephemeral",
				"text":          "Sorry, slash commando, that's an unknown command",
			})
		}
	}
}
