package main

import (
	"os/user"
	"path"

	gomu "github.com/hatchify/mod-utils"
	flag "github.com/hatchify/parg"
	"github.com/hatchify/scribe"
)

var version = "undefined"

func printVersion(cmd *flag.Command) (err error) {
	outW.SetTypePrefix(scribe.TypeNotification, "")
	out.Notification(version)

	return
}

func upgrade(cmd *flag.Command) (err error) {
	var (
		output         string
		version        string
		currentVersion string
		originalBranch string
		headCommit     string
		tagCommit      string
		latestTag      string
		hasChanges     bool
		usr            *user.User
	)

	usr, err = user.Current()
	if err != nil {
		return
	}

	lib := gomu.LibraryFromPath(path.Join(usr.HomeDir, "go", "src", "github.com", "vroomy", "vroomy"))

	if len(cmd.Arguments) > 0 {
		// Set version from args
		if val, ok := cmd.Arguments[0].Value.(string); ok {
			version = val
		} else {
			version = cmd.Arguments[0].Name
		}
	} else {
		version = cmd.StringFrom("-branch")
	}

	lib.File.Output("Checking Installation...")
	currentVersion, _ = lib.File.CmdOutput("vroomy", "version")
	originalBranch, _ = lib.File.CurrentBranch()
	hasChanges = lib.File.HasChanges()
	latestTag = lib.GetLatestTag()

	if len(version) > 0 {
		// Attempt to checkout this version of source
	} else {
		version = latestTag
		if len(currentVersion) > 0 && currentVersion == version {
			if output, err = lib.File.CmdOutput("git", "rev-list", "-n", "1", version); err != nil {
				// No tag set. skip tag
				lib.File.Output("No revision history. Skipping tag.")
				return
			}

			tagCommit = string(output)

			if output, err = lib.File.CmdOutput("git", "rev-parse", "HEAD"); err != nil {
				// No tag set. skip tag
				lib.File.Output("No revision head. Skipping tag.")
				return
			}

			headCommit = string(output)

			if tagCommit == headCommit {
				if hasChanges {
					lib.File.Output("There appears to be local changes...")
				} else {
					lib.File.Output("Version is up to date!")
					return
				}
			} else {
				lib.File.Output("There appears to be an untagged commit...")
			}
		}
	}

	var msg string
	msg = version
	if len(msg) == 0 {
		msg = "latest"
	}

	lib.File.Output("Upgrading Installation from " + currentVersion + " to " + version + "...")

	if len(version) > 0 {
		lib.File.Output("Setting local vroomy repo to: " + version + "...")

		if err = lib.File.CheckoutBranch(version); err != nil {
			lib.File.Output("Failed to checkout " + version + " :(")
			return
		}

		lib.File.Pull()

	} else {
		lib.File.Output("Updating source...")

		if lib.File.Pull() != nil {
			lib.File.Output("Failed to update source :(")
		}
	}

	if hasChanges {
		headCommit = "local"

	} else {
		if tagCommit == "" {
			output, err = lib.File.CmdOutput("git", "rev-list", "-n", "1", version)

			if err != nil {
				// No tag set. skip tag
				lib.File.Output("No revision history. Skipping tag.")

				if len(originalBranch) > 0 {
					lib.File.CheckoutBranch(originalBranch)
				}
				return
			}

			tagCommit = string(output)
		}

		if headCommit == "" {
			output, err = lib.File.CmdOutput("git", "rev-parse", "HEAD")

			if err != nil {
				lib.File.Output("No revision head. Cannot checkout version.")

				if len(originalBranch) > 0 {
					lib.File.CheckoutBranch(originalBranch)
				}
				return
			}

			headCommit = string(output)
		}
	}

	// TODO: Check current tag instead of latest tag?
	if hasChanges || version != latestTag {
		version += "-(" + headCommit + ")"
	}

	if currentVersion == version && tagCommit == headCommit {
		if !hasChanges {
			lib.File.Output("Version is up to date!")

			if len(originalBranch) > 0 {
				lib.File.CheckoutBranch(originalBranch)
			}

			return
		}
	}

	lib.File.Output("Installing " + version + "...")

	if err = lib.File.RunCmd("./install", version); err != nil {
		// Try again with permissions
		err = nil
		if err = lib.File.RunCmd("sudo", "./install", version); err != nil {
			lib.File.Output("Failed to install :(")

			if len(originalBranch) > 0 {
				lib.File.CheckoutBranch(originalBranch)
			}
			return err
		}

		// Fix pkg permission issues
		lib.File.RunCmd("sudo", "chown", "-R", usr.Name, path.Join(usr.HomeDir, "go", "pkg"))
	}

	lib.File.Output("Installed Successfully!")

	if len(originalBranch) > 0 {
		lib.File.CheckoutBranch(originalBranch)
	}

	return
}
