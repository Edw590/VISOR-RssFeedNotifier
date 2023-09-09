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
	"strconv"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"golang.org/x/exp/slices"

	"Utils"
)

const _VID_TIME_DEF string = "--:--"
const _VID_TIME_LIVE string = "00:00" // Live videos have 00:00 as duration

/*
youTubeTreatment processes the YouTube feed.

-----------------------------------------------------------

– Params:
  - feedType – the type of the feed
  - parsed_feed – the parsed feed
  - item_num – the number of the current item in the feed
  - title_url_only – whether to only get the title and URL of the item through _NewsInfo (can be used for optimization)

– Returns:
  - the email info (without the Mail_to field)
  - the news info (useful especially if it's a playlist and it had to be scraped and the item order reversed internally)
All EmailInfo fields are empty if an error occurs, if the video is to be ignored (like if it's a Short), or if
title_url_only is true. In the 1st case, the _NewsInfo fields are also empty. In the 2nd case, the _NewsInfo fields are
still filled with the video info. To check for errors, check if the video URL is empty on NewsInfo (that one must always
have a value).
*/
func youTubeTreatment(feedType _FeedType, parsed_feed *gofeed.Feed, item_num int, title_url_only bool) (Utils.EmailInfo,
			_NewsInfo) {
	const (
		VIDEO_COLOR string = "#212121; padding" // Default video color (sort of black)
		LIVE_COLOR  string = "#E62117; padding" // Default live color (sort of red)
	)
	const (
		CH_ID      string = "|3234_CHANNEL_CODE|"
		CH_NAME    string = "|3234_CHANNEL_NAME|"
		CH_IMAGE   string = "|3234_CHANNEL_IMAGE|"
		PL_ID      string = "|3234_PLAYLIST_CODE|"
		VID_TITLE  string = "|3234_VIDEO_TITLE|"
		VID_ID     string = "|3234_VIDEO_CODE|"
		VID_IMAGE  string = "|3234_VIDEO_IMAGE|"
		VID_DESC   string = "|3234_VIDEO_DESCRIPTION|"
		VID_LEN    string = "|3234_VIDEO_TIME|"
		SUB_NAME   string = "|3234_SUBSCRIPTION_NAME|"
		SUB_LINK   string = "|3234_SUBSCRIPTION_LINK|"
		TIME_COLOR string = VIDEO_COLOR
		HTML_TITLE string = "|3234_HTML_TITLE|"
	)

	var things_replace = map[string]string{
		CH_NAME:    parsed_feed.Authors[0].Name,
		CH_ID:      parsed_feed.Items[0].Extensions["yt"]["channelId"][0].Value,
		CH_IMAGE:   _GEN_ERROR,
		PL_ID:      "", // Leave empty if it's not playlist
		VID_TITLE:  _GEN_ERROR,
		VID_ID:     _GEN_ERROR,
		VID_IMAGE:  _GEN_ERROR,
		VID_DESC:   _GEN_ERROR,
		VID_LEN:    _GEN_ERROR,
		SUB_NAME:   parsed_feed.Title,
		SUB_LINK:   _GEN_ERROR,
		TIME_COLOR: VIDEO_COLOR,
		HTML_TITLE: _GEN_ERROR,
	}
	if !title_url_only {
		things_replace[CH_IMAGE] = getChannelImageUrl(things_replace[CH_ID])
	}

	if feedType.type_2 == _TYPE_2_YT_CHANNEL {
		// The last part is what YouTube used to put in the URLs (taken from the original model)
		things_replace[SUB_LINK] = "channel/" + things_replace[CH_ID] + "%3Ffeature%3Dem-uploademail"
	} else if feedType.type_2 == _TYPE_2_YT_PLAYLIST {
		things_replace[PL_ID] = parsed_feed.Extensions["yt"]["playlistId"][0].Value
		things_replace[SUB_LINK] = "playlist?list=" + things_replace[PL_ID]
	}

	if feedType.type_2 == _TYPE_2_YT_PLAYLIST && scrapingNeeded(parsed_feed) {
		// Scraping is only needed for video information. The feed has the rest.
		// For scraping we only use the number of the item to guide through the video array. The rest comes from the
		// playlist page.
		var video_info _VideoInfo = ytPlaylistScraping(things_replace[PL_ID], item_num, len(parsed_feed.Items))
		if video_info.id == _GEN_ERROR {
			return Utils.EmailInfo{}, _NewsInfo{}
		}

		things_replace[VID_TITLE] = video_info.title
		things_replace[VID_ID] = video_info.id
		things_replace[VID_IMAGE] = video_info.image
		things_replace[VID_LEN] = video_info.length

		// No way to get the description from the playlist visual page unless the video appears on the RSS feed.
		things_replace[VID_DESC] = _GEN_ERROR
		for _, item := range parsed_feed.Items {
			if item.Extensions["yt"]["videoId"][0].Value == video_info.id {
				things_replace[VID_DESC] = item.Extensions["media"]["group"][0].Children["description"][0].Value

				break
			}
		}
	} else {
		var feed_item *gofeed.Item = parsed_feed.Items[item_num]
		things_replace[VID_TITLE] = feed_item.Title
		things_replace[VID_ID] = feed_item.Extensions["yt"]["videoId"][0].Value
		things_replace[VID_IMAGE] = feed_item.Extensions["media"]["group"][0].Children["thumbnail"][0].Attrs["url"]
		things_replace[VID_DESC] = feed_item.Extensions["media"]["group"][0].Children["description"][0].Value
		if !title_url_only {
			things_replace[VID_LEN] = getVideoDuration(feed_item.Link)
		}
	}

	var is_short bool = isShort([]string{things_replace[VID_TITLE], things_replace[VID_DESC]}, things_replace[VID_LEN])

	// If it's not to include Shorts and the video is a Short, return only the news info (to ignore the notification but
	// memorize that the video is to be ignored).
	if (feedType.type_3 != _TYPE_3_YT_INC_SHORTS && is_short) || title_url_only {
		return Utils.EmailInfo{}, _NewsInfo{
			title: things_replace[VID_TITLE],
			url: "https://www.youtube.com/watch?v=" + things_replace[VID_ID],
		}
	}

	// Like YouTube used to do - trim after 67 chars
	var vid_title string = things_replace[VID_TITLE]
	var vid_title_original string = vid_title
	if len(vid_title) > 67 {
		vid_title = vid_title[:67] + "..."
		things_replace[VID_TITLE] = vid_title
	}

	var video_short string = ""
	if is_short {
		video_short = "Short"
	} else {
		video_short = "vídeo"
	}

	var msg_subject string = _GEN_ERROR
	if feedType.type_2 == _TYPE_2_YT_CHANNEL {
		if _VID_TIME_LIVE == things_replace[VID_LEN] {
			// Live video
			msg_subject = "🔴 " + things_replace[CH_NAME] + " está agora em direto: " + vid_title + "!"
			things_replace[HTML_TITLE] = "Em direto no YouTube: " + things_replace[CH_NAME] + " – " + vid_title + "!"

			// Change the length rectangle
			things_replace[TIME_COLOR] = LIVE_COLOR
			things_replace[VID_LEN] = "LIVE" // Change the video length to "LIVE"
		} else {
			// Normal video
			msg_subject = things_replace[CH_NAME] + " acabou de carregar um " + video_short
			things_replace[HTML_TITLE] = msg_subject
		}
	} else if feedType.type_2 == _TYPE_2_YT_PLAYLIST {
		// Playlist video
		msg_subject = things_replace[CH_NAME] + " acabou de adicionar um " + video_short + " a " + parsed_feed.Title
		things_replace[HTML_TITLE] = msg_subject
	}

	if len(things_replace[VID_DESC]) > 67 {
		// The original was up to 27 chars but I increased it to the same as the title to appear instead of "RSS Feed
		// notifications" in the email preview.
		things_replace[VID_DESC] = things_replace[VID_DESC][:67] + "..."
	}

	var msg_html string = *Utils.GetModelFileEMAIL(Utils.MODEL_FILE_YT_VIDEO)
	for key, value := range things_replace {
		msg_html = strings.ReplaceAll(msg_html, key, value)
	}

	return Utils.EmailInfo{
		Sender:  "YouTube",
		Subject: msg_subject,
		Html:    msg_html,
	},
	_NewsInfo{
		title: vid_title_original,
		url:   "https://www.youtube.com/watch?v=" + things_replace[VID_ID],
	}
}

/*
getVideoDuration gets the duration of the video by getting the video's page and looking for the duration (scraping).

The format returned is the same as the one from the SecondsToTimeStr() function.

-----------------------------------------------------------

– Params:
  - video_url – the URL of the video

– Returns:
  - the duration of the video if it was found, _VID_TIME_DEF otherwise
*/
func getVideoDuration(video_url string) string {
	var p_page_html *string = Utils.GetPageHtmlTIMEDATE(video_url)
	if nil == p_page_html {
		return _VID_TIME_DEF
	}
	var page_html string = *p_page_html

	// I think the data is in JSON, so I got the lengthSeconds that I found randomly looking for the seconds. It also a
	// double quote after the number ("lengthSeconds":"47" for 47 seconds) --> CAN CHANGE (checked on 2023-07-04).
	text_to_find := "\"lengthSeconds\":\""
	idx_begin := strings.Index(page_html, text_to_find) + len(text_to_find)
	idx_end := strings.Index(page_html[idx_begin:], "\"")
	if idx_begin > 0 && idx_end > 0 {
		return SecondsToTimeStr(page_html[idx_begin : idx_begin+idx_end])
	}

	return _VID_TIME_DEF
}

/*
SecondsToTimeStr converts the seconds to a time string for the video duration.

The format returned is "HH:MM:SS", but if the video is less than an hour long, the hours are removed ("MM:SS").

-----------------------------------------------------------

– Params:
  - seconds_str – the number of seconds as a string

– Returns:
  - the time string
 */
func SecondsToTimeStr(seconds_str string) string {
	var seconds, _ = strconv.Atoi(seconds_str)
	// Note: the location here is useless - I need a duration, not a date. So I chose UTC because yes.
	var length_seconds_time = time.Date(0, 0, 0, 0, 0, seconds, 0, time.UTC)
	var time_str string = length_seconds_time.Format("15:04:05")
	if strings.HasPrefix(time_str, "00:") {
		// Remove the hours if the video is less than an hour long
		time_str = time_str[3:]
	}

	return time_str
}

/*
getChannelImageUrl gets the URL of the channel image of by getting the channel's page and looking for the image
(scraping).

-----------------------------------------------------------

– Params:
  - channel_code – the code of the channel

– Returns:
  - the URL of the channel image if it was found, _GEN_ERROR otherwise
*/
func getChannelImageUrl(channel_code string) string {
	var p_page_html *string = Utils.GetPageHtmlTIMEDATE("https://www.youtube.com/channel/" + channel_code)
	if nil == p_page_html {
		return _GEN_ERROR
	}
	var page_html string = *p_page_html

	// The image URL is on the 3rd occurrence of the "https://yt3.googleusercontent.com/" on HTML of the channel's page
	// --> CAN CHANGE (checked on 2023-07-04).
	// The 1st and 2nd occurrences are the user's image and the channel's background image, respectively.
	var text_to_find string = "https://yt3.googleusercontent.com/"
	var idxs_begin []int = Utils.FindAllIndexesGENERAL(page_html, text_to_find)
	if len(idxs_begin) >= 3 {
		var idx_begin int = idxs_begin[2]
		var idx_end int = strings.Index(page_html[idx_begin:], "\"")

		return page_html[idx_begin : idx_begin+idx_end]
	}

	return _GEN_ERROR
}

/*
isShort checks if the video is a Short.

-----------------------------------------------------------

– Params:
  - video_texts – the texts of the video like title and description
  - video_len – the length of the video from getVideoDuration()

– Returns:
  - true if the video is a short, false otherwise (also false if video_len is _VID_TIME_DEF)
 */
func isShort(video_texts []string, video_len string) bool {
	// If any of the video texts has the #short or #shorts tag, mark as Short.
	for _, video_text := range video_texts {
		video_text_words := strings.Split(strings.ToLower(video_text), " ")
		if slices.Contains(video_text_words, "#short") || slices.Contains(video_text_words, "#shorts") {
			return true
		}
	}

	if _VID_TIME_DEF == video_len {
		// If the video length was not found, mark as not Short (can't know, better safe than sorry).
		return false
	}

	// Lastly, if none of the others worked (a video can be a Short and not have the tags), if the video is 1 minute or
	// less long, mark it as Short.
	var length_seconds = 0
	if len(Utils.FindAllIndexesGENERAL(video_len, ":")) == 1 {
		length_parsed, _ := time.Parse("04:05", video_len)
		length_seconds = length_parsed.Minute()*60 + length_parsed.Second()
	} else {
		length_parsed, _ := time.Parse("15:04:05", video_len)
		length_seconds = length_parsed.Hour()*60*60 + length_parsed.Minute()*60 + length_parsed.Second()
	}

	return length_seconds <= 60
}
