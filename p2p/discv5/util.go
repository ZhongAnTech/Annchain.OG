package discv5

func topicsToStrings(topics []Topic) []string {
	var topicsString []string
	for _, v := range topics {
		topicsString = append(topicsString, string(v))
	}
	return topicsString
}

func stringsToTopics(topicsString []string) Topics {
	var topics []Topic
	for _, v := range topicsString {
		topics = append(topics, Topic(v))
	}
	return topics
}
