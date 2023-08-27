/*******************************************************************************
 * Copyright 2023-2023 Edw590
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 ******************************************************************************/

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"VISOR_Server/Utils"
)

// RSS Feed Notifier //

// Cut the video title in more than 67 characters. After those 67 put ellipsis. This is what YT used to do.
// //////////////////////////////////////////////////
// (The emails are in PT-PT)
// Família Brisados acabou de carregar um vídeo
// 🔴 SuperHouseTV está agora em direto: [video title here]
// //////////////////////////////////////////////////

//////////////////////////
// Types of feeds:
var allowed_feed_types_1_GL []string = []string{
	_TYPE_1_GENERAL,
	_TYPE_1_YOUTUBE,
}
const (
	_TYPE_1_GENERAL = "General"
	_TYPE_1_YOUTUBE = "YouTube"
)
const (
	_TYPE_2_YT_CHANNEL  = "CH"
	_TYPE_2_YT_PLAYLIST = "PL"
)
const (
	_TYPE_3_YT_INC_SHORTS = "+S"
)
//////////////////////////

const _GEN_ERROR string = "3234_ERROR"

// _FeedType is the type of the feed. Each type is one of the TYPE_x constants, being x the number of the type.
type _FeedType struct {
	type_1 string
	type_2 string
	type_3 string
}

// _NewsInfo is the information about news.
type _NewsInfo struct {
	url string
	title string
}

// _MAX_URLS_STORED is the maximum number of URLs stored in the file. This is to avoid having a file with too many URLs.
// 100 because it must be above the number of entries in all the feeds, and 100 is a big number (30 for StackExchange,
// 15 for YT - 100 seems perfect).
const _MAX_URLS_STORED int = 100

type _ModSpecificInfo any
var (
	realMain Utils.RealMain = nil
	modProvInfo_GL Utils.ModProvInfo
	modGenFileInfo_GL Utils.ModGenFileInfo[_ModSpecificInfo]
)
func main() {Utils.ModStartup[_ModSpecificInfo](Utils.NUM_MOD_RssFeedNotifier, realMain)}
func init() {realMain =
	func(realMain_param_1 Utils.ModProvInfo, realMain_param_2 any) {
		modProvInfo_GL = realMain_param_1
		modGenFileInfo_GL = realMain_param_2.(Utils.ModGenFileInfo[_ModSpecificInfo])

		for {
			var feedsInfo []_MFIFeedInfo = getFeedsInfo()
			if nil == feedsInfo {
				fmt.Println("Error getting feeds info")

				goto end_loop
			}

			for _, feedInfo := range getFeedsInfo() {
				// if 8 != feedInfo.Feed_num {
				//	continue
				// }
				fmt.Println("__________________________BEGINNING__________________________")

				var feedType _FeedType = getFeedType(feedInfo.Feed_type)

				if !Utils.ContainsSLICES(allowed_feed_types_1_GL, feedType.type_1) {
					fmt.Println("Feed type not allowed: " + feedInfo.Feed_type)
					fmt.Println("__________________________ENDING__________________________")

					continue
				}

				if _TYPE_1_YOUTUBE == feedType.type_1 {
					// If the feed is a YouTube feed, the feed URL is the channel or playlist ID, so we need to change it to
					// the correct URL.
					if _TYPE_2_YT_CHANNEL == feedType.type_2 {
						feedInfo.Feed_url = "https://www.youtube.com/feeds/videos.xml?channel_id=" + feedInfo.Feed_url
					} else if _TYPE_2_YT_PLAYLIST == feedType.type_2 {
						feedInfo.Feed_url = "https://www.youtube.com/feeds/videos.xml?playlist_id=" + feedInfo.Feed_url
					}
				}

				fmt.Println("feed_num: " + strconv.Itoa(feedInfo.Feed_num))
				fmt.Println("feed_url: " + feedInfo.Feed_url)
				fmt.Println("feed_type: " + feedInfo.Feed_type)
				fmt.Println("feedType.type_1: " + feedType.type_1)
				fmt.Println("feedType.type_2: " + feedType.type_2)
				fmt.Println("feedType.type_3: " + feedType.type_3)

				var notif_news_file_path Utils.GPath = modProvInfo_GL.Data_dir.Add("urls_notified_news/",
					strconv.Itoa(feedInfo.Feed_num) + ".txt")
				var newsInfo_list []_NewsInfo = nil
				var notified_news_list []string = nil
				if notif_news_file_path.Exists() {
					newsInfo_list = make([]_NewsInfo, 0, _MAX_URLS_STORED)
					var notified_news string = *notif_news_file_path.ReadFile()
					notified_news_list = strings.Split(notified_news, "\n")
					for _, line := range notified_news_list {
						var line_split []string = strings.Split(line, " \\\\// ")
						if 2 == len(line_split) {
							newsInfo_list = append(newsInfo_list, _NewsInfo{
								url:   line_split[0],
								title: line_split[1],
							})
						}
					}
				}

				var new_feed bool = false
				if 0 == len(newsInfo_list) {
					new_feed = true
				}

				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				parsed_feed, err := gofeed.NewParser().ParseURLWithContext(feedInfo.Feed_url, ctx)
				cancel()
				if nil != err {
					fmt.Println("Error parsing feed: " + err.Error())
					continue
				}

				var notified_news_list_modified bool = false
				for item_num, item := range parsed_feed.Items {

					var check_skipping_later bool = true

					// Check if the news is new, and if it's not, skip it. But only if it's not a YouTube playlist, because
					// those may need the order of the items reversed and so the ones got from this loop are wrong. Or if it
					// is playlist, then only if the feed item ordering is correct (no scraping needed).
					// This is also here and not just in the end to prevent useless item processing (optimized).
					if _TYPE_2_YT_PLAYLIST != feedType.type_2 || !scrapingNeeded(parsed_feed) {
						check_skipping_later = false
						if !isNewNews(newsInfo_list, item.Title, item.Link) {
							// If the news is not new, don't notify.
							continue
						}
					}

					var email_info Utils.EmailInfo = Utils.EmailInfo{}
					var newsInfo _NewsInfo = _NewsInfo{}

					switch feedType.type_1 {
						case _TYPE_1_YOUTUBE: {
							email_info, newsInfo = youTubeTreatment(feedType, parsed_feed, item_num, new_feed)
						}
						case _TYPE_1_GENERAL: {
							email_info, newsInfo = generalTreatment(parsed_feed, item_num, new_feed,
								feedInfo.Custom_msg_subject)
						}
						default: {
							fmt.Println("Unknown feed type_1: " + feedType.type_1)
							continue
						}
					}

					var ignore_video bool = "" == email_info.Html

					if "" == newsInfo.url { // Some error occurred
						continue
					}

					if check_skipping_later && !isNewNews(newsInfo_list, newsInfo.title, newsInfo.url) {
						// If the news is not new, don't notify.
						continue
					}

					var error_notifying bool = false

					fmt.Println("New news: " + newsInfo.title)
					if !new_feed && !ignore_video {
						// If the feed is a newly added one, don't send emails for ALL the items in the feed - which are
						// being treated for the first time.
						fmt.Println("Queuing email: " + email_info.Subject + " by " + email_info.Sender)
						error_notifying = !queueEmailAllRecps(email_info.Sender, email_info.Subject, email_info.Html)
					}

					if !error_notifying {
						notified_news_list = append(notified_news_list, newsInfo.url+" \\\\// "+newsInfo.title)
						if len(notified_news_list) > _MAX_URLS_STORED {
							notified_news_list = notified_news_list[1:]
						}
						notified_news_list_modified = true
					}
				}
				if notified_news_list_modified {
					notif_news_file_path.WriteTextFile(strings.Join(notified_news_list, "\n"))
				}

				fmt.Println("__________________________ENDING__________________________")
			}

			end_loop:

			return

			modGenFileInfo_GL.LoopSleep(2*60)
		}
	}
}

/*
getFeedType gets the _FeedType information from _MFIFeedInfo.Feed_type.

-----------------------------------------------------------

– Params:
  - feed_type – the _MFIFeedInfo.Feed_type

– Returns:
  - the _FeedType information
*/
func getFeedType(feed_type string) _FeedType {
	var feed_type_split []string = strings.Split(feed_type, " ")
	var feed_type_split_len int = len(feed_type_split)
	var feedType _FeedType = _FeedType{}
	if feed_type_split_len >= 1 {
		feedType.type_1 = feed_type_split[0]
	}
	if feed_type_split_len >= 2 {
		feedType.type_2 = feed_type_split[1]
	}
	if feed_type_split_len >= 3 {
		feedType.type_3 = feed_type_split[2]
	}

	return feedType
}

/*
isNewNews checks if the news is new.

-----------------------------------------------------------

– Params:
  - newsInfo_list – the list of notified news
  - title – the title of the news
  - url – the URL of the news

– Returns:
  - true if the news is new, false otherwise
 */
func isNewNews(newsInfo_list []_NewsInfo, title string, url string) bool {
	fmt.Println("Checking if news is new: " + title)
	for _, newsInfo := range newsInfo_list {
		if  newsInfo.url == url && newsInfo.title == title {
			return false
		}
	}

	fmt.Println("News is new ^^^^^")

	return true
}

/*
getFeedsInfo gets the information of the feeds.

-----------------------------------------------------------

– Returns:
  - the information of the feeds or nil if an error occurs
*/
func getFeedsInfo() []_MFIFeedInfo {
	var modFileInfo _ModFileInfo
	if !modProvInfo_GL.GetModUserInfo(&modFileInfo) {
		return nil
	}

	return modFileInfo.Feeds_info
}

/*
queueEmailAllRecps queues an email to be sent to all recipients.

-----------------------------------------------------------

– Params:
  - sender_name – the name of the sender
  - subject – the subject of the email
  - html – the HTML of the email
 */
func queueEmailAllRecps(sender_name string, subject string, html string) bool {
	var modFileInfo _ModFileInfo
	if !modProvInfo_GL.GetModUserInfo(&modFileInfo) {
		return false
	}

	for _, mail_to := range modFileInfo.Mails_to {
		// This is to add the images to the email using CIDs instead of using URLs which could/can go down at any time.
		// Except most email clients don't support CIDs... So I'll leave this here in case the images stop working with
		// the URLs and then either this or embeded Base64 on the src attribute of the <img> tag or hosted in the server
		// or something.
		// Still, the CID way seems better than the Base64 one. With CIDs, only Gmail Notified Pro wasn't showing them.
		// With Base64, Gmail (web or app) wasn't showing them (don't remember about the notifier). But I didn't test in
		// Hotmail or others.
		//var multiparts []Utils.Multipart = nil
		//if youtube {
		//	// Add the YouTube images to the email instead of using URLs which could/can go down at any time.
		//	var files_add []string = []string{"transparent_pixel.png", "twitter_email_icon_grey.png",
		//		"youtube_email_icon_grey.png", "youtubelogo_60.png"}
		//	for _, file_add := range files_add {
		//		var multipart Utils.Multipart = Utils.Multipart{
		//			Content_type: "image/png",
		//			Content_transfer_encoding: "base64",
		//			Content_id: file_add,
		//		}
		//		data, _ := os.ReadFile(modProvInfo_GL.Dir.Add("yt_email_images/", file_add).
		//			GPathToStringConversion())
		//		multipart.Body = base64.StdEncoding.EncodeToString(data)
		//
		//		multiparts = append(multiparts, multipart)
		//	}
		//}

		// Write the HTML to a file in case debugging is needed.
		modProvInfo_GL.Temp_dir.Add("last_html_queued.html").WriteTextFile(html)

		Utils.QueueEmailEMAIL(Utils.EmailInfo{
			Sender:  sender_name,
			Mail_to: mail_to,
			Subject: subject,
			Html:    html,
			Multiparts: nil,
		})
	}

	return true
}
