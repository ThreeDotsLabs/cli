package trainings

type UserFacingError struct {
	Msg          string
	SolutionHint string
}

func (u UserFacingError) Error() string {
	return u.Msg + " " + u.SolutionHint
}
