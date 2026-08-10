package models

type User struct {
	Name            string   `json:"name"`
	Email           string   `json:"email"`
	Age             int      `json:"age"`
	FavouriteTopics []string `json:"favouriteTopics"`
}
