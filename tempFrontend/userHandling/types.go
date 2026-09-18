package userhandling

// UpdateAccountData carries the pre-populated user account fields that the
// update account page renders server-side (name + which topics are selected).
type UpdateAccountData struct {
	Name            string
	FavouriteTopics map[string]bool
}
