package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"charm.land/huh/v2"
)

// ProfileFormData is used by interactive add.
type ProfileFormData struct {
	Profile      string
	IdentityFile string
	GithubUser   string
	HostAlias    string
}

// SelectProfile prompts for a profile name from the list.
func SelectProfile(names []string, in io.Reader, out io.Writer) (string, error) {
	if len(names) == 0 {
		return "", fmt.Errorf("there are no available profiles")
	}

	var profile string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a profile").
				Options(huh.NewOptions(names...)...).
				Value(&profile),
		),
	).WithInput(in).WithOutput(out)

	if err := form.Run(); err != nil {
		return "", err
	}
	return profile, nil
}

// PromptProfileName asks for an existing or new profile name.
func PromptProfileName(in io.Reader, out io.Writer) (string, error) {
	var profile string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter an existing profile name or a new one.").
				Value(&profile).
				Validate(func(input string) error {
					if strings.TrimSpace(input) == "" {
						return fmt.Errorf("profile name cannot be empty")
					}
					return nil
				}),
		),
	).WithInput(in).WithOutput(out)

	if err := form.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(profile), nil
}

// PromptProfileFields edits SSH profile fields (prefilled when updating).
func PromptProfileFields(initial ProfileFormData, in io.Reader, out io.Writer) (ProfileFormData, error) {
	result := initial
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("IdentityFile (empty = auto ~/.ssh/git-ssh/<profile>/id_ed25519)").Value(&result.IdentityFile),
			huh.NewInput().Title("GitHub user/owner for origin (empty = profile name)").Value(&result.GithubUser),
			huh.NewInput().Title("Optional Host alias (not github.com)").Value(&result.HostAlias),
		),
	).WithInput(in).WithOutput(out)

	if err := form.Run(); err != nil {
		return ProfileFormData{}, err
	}
	result.IdentityFile = strings.TrimSpace(result.IdentityFile)
	result.GithubUser = strings.TrimSpace(result.GithubUser)
	result.HostAlias = strings.TrimSpace(result.HostAlias)
	return result, nil
}

// Confirm asks a yes/no question and reports the result.
func Confirm(title string, affirmative, negative string, in io.Reader, out io.Writer) (bool, error) {
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative(affirmative).
				Negative(negative).
				Value(&confirmed),
		),
	).WithInput(in).WithOutput(out)

	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}

// IsAborted reports huh user cancel.
func IsAborted(err error) bool {
	return errors.Is(err, huh.ErrUserAborted)
}
