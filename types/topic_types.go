package types

type TopicType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var TopicTypes = []TopicType{
	{ID: 1, Name: "notebook"},
	{ID: 2, Name: "dashboard"},
}

func GetTopicTypeByID(id int) *TopicType {
	for _, t := range TopicTypes {
		if t.ID == id {
			return &t
		}
	}
	return nil
}

func GetTopicTypeByName(name string) *TopicType {
	for _, t := range TopicTypes {
		if t.Name == name {
			return &t
		}
	}
	return nil
}
