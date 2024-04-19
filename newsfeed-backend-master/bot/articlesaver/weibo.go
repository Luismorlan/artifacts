package articlesaver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/dstotijn/go-notion"
)

// this request is generated using https://mholt.github.io/curl-to-go/ with signed in tokens
func GetWeiboArticle(url string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Cookie", "PC_TOKEN=525371698d; WBStorage=4d96c54e|undefined; WBPSESS=WBdCd2lKXvNjNFUrk40JEUaoneZKEiRzUTn4kXpCZ2Z8PhCpbSQz4p7zGGciiXx34RRq27x0wPCWehCN6yaf5F31VbIfxtAPoWR8ktPmd0l4VZym-0eglQcTIODHye21De973PHuyz6p-_UrL9Olrw==; ALF=1725899475; SUB=_2A25J-Z8FDeRhGeNG7lER8SjFzTyIHXVqjvfNrDV8PUNbmtAbLVfAkW9NS1dszoabZ0Eu7gP1_FE8Zp7GKzL3fQf9; SUBP=0033WrSXqPxfM725Ws9jqgMF55529P9D9W5wYRUgbf8D2UvfSyzwQ6PR5JpX5o275NHD95Qf1h-0eh2c1Kq7Ws4DqcjG-2yheo5Ee7tt; SSOLoginState=1694363476; cross_origin_proto=SSL; Apache=9052549790116.635.1694363070820; SINAGLOBAL=9052549790116.635.1694363070820; ULV=1694363070824:1:1:1:9052549790116.635.1694363070820:; wb_view_log=1920*10802; _s_tentry=weibo.com; login_sid_t=6332d0b2ac93e7d6479c32003f0bc82d; XSRF-TOKEN=9EMvPfqT3q2h3cH5dvZxYwY-")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15")
	req.Header.Set("Referer", "https://weibo.com/u/2453509265?tabtype=article")
	req.Header.Set("Connection", "keep-alive")

	res, error := http.DefaultClient.Do(req)
	if error != nil {
		return nil, error
	}
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Error code %d", res.StatusCode)
	}
	return goquery.NewDocumentFromReader(res.Body)
}

func SaveWeiboDocToNotion(doc *goquery.Document, url string) (string, error) {
	client := NewNotionClient()
	ctx := context.Background()
	title := doc.Find("title").Text()
	blocks := []notion.Block{}
	doc.Find(".WB_editor_iframe_new").Each(func(i int, articleDiv *goquery.Selection) {
		for _, kk := range articleDiv.Children().Nodes {
			if kk.Data == "figure" {
				blocks = append(blocks, notion.ImageBlock{
					Type: notion.FileTypeExternal,
					External: &notion.FileExternal{
						URL: kk.FirstChild.Attr[0].Val,
					},
				})
			}
			if kk.Data == "p" {
				if kk.FirstChild.Data == "strong" {
					blocks = append(blocks, notion.ParagraphBlock{
						RichText: []notion.RichText{
							{
								Text: &notion.Text{
									Content: kk.FirstChild.FirstChild.Data,
								},
								Annotations: &notion.Annotations{
									Bold: true,
								}},
						},
					})
				} else {
					if len(strings.TrimSpace(kk.FirstChild.Data)) > 0 {
						blocks = append(blocks, notion.ParagraphBlock{
							RichText: []notion.RichText{
								{
									Text: &notion.Text{
										Content: kk.FirstChild.Data,
									},
								},
							}})
					}
				}
			}
		}
	})

	blocks = append(blocks, notion.ParagraphBlock{
		RichText: []notion.RichText{
			{
				Text: &notion.Text{
					Content: "原文链接",
					Link: &notion.Link{
						URL: url,
					},
				},
			},
		},
	})
	params := notion.CreatePageParams{
		ParentType: notion.ParentTypePage,
		ParentID:   NOTION_PAGE_ID,
		Title: []notion.RichText{
			{
				Text: &notion.Text{
					Content: title,
				},
			},
		},
		Children: blocks,
	}
	page, err := client.CreatePage(ctx, params)
	if err != nil {
		log.Fatalf("Failed to create page: %v", err)
	}
	return page.URL, nil
}
