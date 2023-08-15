package main

// _ModFileInfo is the format of the custom information file about this specific module.
type _ModFileInfo struct {
	// Mails_info is the information about the mails to send the feeds info to
	Mails_to   []string
	// Feed_info is the information about the feeds
	Feeds_info []_MFIFeedInfo
}

// _MFIFeedInfo is the information about a feed.
type _MFIFeedInfo struct {
	// Feed_num is the number of the feed, beginning in 1 (no special reason, but could be useful some time)
	Feed_num int
	// Feed_url is the URL of the feed
	Feed_url string
	// Feed_type is the type of the feed (one of the TYPE_ constants)
	Feed_type string
	// Custom_msg_subject is the custom message subject
	Custom_msg_subject string
}
