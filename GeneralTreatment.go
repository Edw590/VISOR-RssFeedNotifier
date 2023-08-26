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
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"VISOR_S_L/Utils"
)

/*
generalTreatment does the general treatment of an RSS feed item.

-----------------------------------------------------------

– Params:
  - parsed_feed – the parsed feed
  - item_num – the number of the item to get
  - title_url_only – whether to only get the title and URL of the item through _NewsInfo (can be used for optimization)

– Returns:
  - the email info (without the Mail_to field) or all fields empty if title_url_only is true
  - the news info
 */
func generalTreatment(parsed_feed *gofeed.Feed, item_num int, title_url_only bool, custom_msg_subject string) (
					Utils.EmailInfo, _NewsInfo) {
	const (
		TITLE      string = "|3234_ENTRY_TITLE|"
		URL        string = "|3234_ENTRY_URL|"
		AUTHOR     string = "|3234_ENTRY_AUTHOR|"
		AUTHOR_URL string = "|3234_ENTRY_AUTHOR_URL|"
		PUB_DATE   string = "|3234_ENTRY_PUB_DATE|"
		UPD_DATE   string = "|3234_ENTRY_UPD_DATE|"
		DESC       string = "|3234_ENTRY_DESCRIPTION|"
	)

	var feed_item *gofeed.Item = parsed_feed.Items[item_num]

	var things_replace = map[string]string{
		TITLE:      feed_item.Title,
		URL:        feed_item.Link,
		AUTHOR:     feed_item.Authors[0].Name,
		AUTHOR_URL: "", // Can't get with gofeed
		PUB_DATE:   feed_item.Published,
		UPD_DATE:   feed_item.Updated,
		DESC:       feed_item.Description,
	}
	var newsInfo _NewsInfo = _NewsInfo{
		title: things_replace[TITLE],
		url:   things_replace[URL],
	}

	if title_url_only {
		return Utils.EmailInfo{}, newsInfo
	}

	if things_replace[UPD_DATE] != "" {
		if things_replace[UPD_DATE] == things_replace[PUB_DATE] {
			things_replace[UPD_DATE] = "[new]"
		}
	}

	things_replace[PUB_DATE] = convertDate(things_replace[PUB_DATE])
	things_replace[UPD_DATE] = convertDate(things_replace[UPD_DATE])

	var msg_html string = *Utils.GetModelFileEMAIL(Utils.MODEL_FILE_RSS)
	for key, value := range things_replace {
		msg_html = strings.ReplaceAll(msg_html, key, value)
	}

	return Utils.EmailInfo{
		Sender:  "VISOR - RSS",
		Subject: custom_msg_subject,
		Html:    msg_html,
	}, newsInfo
}

/*
convertDate converts a date from RFC3339 to DATE_FORMAT, also correcting the timezone to the local one.

-----------------------------------------------------------

– Params:
  - date – the date to convert

– Returns:
  - the converted date or the original date if it couldn't be converted
 */
func convertDate(date string) string {
	var date_time, err = time.Parse(time.RFC3339, date)
	if nil != err {
		return date
	}

	return date_time.Local().Format(Utils.DATE_TIME_FORMAT)
}
