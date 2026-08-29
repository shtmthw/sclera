package models

type User struct {
	Name            string   `json:"name"`
	Email           string   `json:"email"`
	Age             int      `json:"age"`
	Password        string   `json:"password"`
	FavouriteTopics []string `json:"favouriteTopics"`
}
type LoggedUser struct {
	Id       int
	Password string
}
type UpdatedUser struct {
	Id              int      `json:"id"`
	Name            string   `json:"name"`
	FavouriteTopics []string `json:"favouriteTopics"`
}
