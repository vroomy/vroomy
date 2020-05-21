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
	lib.File.Fetch()

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

	out.Notification("Checking vroomy installation...")
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
				out.Notification("No revision history. Skipping tag.")
				return
			}

			tagCommit = string(output)

			if output, err = lib.File.CmdOutput("git", "rev-parse", "HEAD"); err != nil {
				// No tag set. skip tag
				out.Notification("No revision head. Skipping tag.")
				return
			}

			headCommit = string(output)

			if tagCommit == headCommit {
				if hasChanges {
					out.Notification("There appears to be local changes...")
				} else {
					out.Successf("%s is up to date!", version)
					return
				}
			} else {
				out.Notification("There appears to be an untagged commit...")
			}
		}
	}

	var msg string
	msg = version
	if len(msg) == 0 {
		msg = "latest"
	}

	if hasChanges {
		msg += " with local changes"
	}

	out.Notification("Upgrading Installation from " + currentVersion + " to " + msg + "...")

	if len(version) > 0 {
		out.Notification("Setting local vroomy repo to: " + version + "...")

		if err = lib.File.CheckoutBranch(version); err != nil {
			out.Notification("Failed to checkout " + version + " :(")
			return
		}

		lib.File.Pull()

	} else {
		out.Notification("Updating source...")

		if lib.File.Pull() != nil {
			out.Notification("Failed to update source :(")
		}
	}

	if hasChanges {
		headCommit = "local"

	} else {
		if tagCommit == "" {
			output, err = lib.File.CmdOutput("git", "rev-list", "-n", "1", version)

			if err != nil {
				// No tag set. skip tag
				out.Notification("No revision history. Skipping tag.")

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
				out.Error("No revision head. Cannot checkout version.")

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
			out.Successf("%s is up to date!", version)

			if len(originalBranch) > 0 {
				lib.File.CheckoutBranch(originalBranch)
			}

			return
		}
	}

	out.Notification("Installing " + version + "...")

	if err = lib.File.RunCmd("./bin/install", version); err != nil {
		// Try again with permissions
		err = nil
		if err = lib.File.RunCmd("sudo", "./bin/install", version); err != nil {
			out.Notification("Failed to install :(")

			if len(originalBranch) > 0 {
				lib.File.CheckoutBranch(originalBranch)
			}
			return
		}

		// Fix pkg permission issues
		lib.File.RunCmd("sudo", "chown", "-R", usr.Name, path.Join(usr.HomeDir, "go", "pkg"))
	}

	var setcap = false
	var homeDir = usr.HomeDir
	if lib.File.RunCmd("which", "setcap") == nil {
		// We can setcap! Let's move to /usr/local/bin and run setcap
		if err = lib.File.RunCmd("sudo", "mv", path.Join(homeDir, "go", "bin", "vroomy"), "/usr/local/bin/vroomy"); err != nil {
			out.Warningf("Unable to move vroomy to /usr/local/bin: %v", err)
		} else if err = lib.File.RunCmd("sudo", path.Join(lib.File.AbsPath(), "bin/setcap"), "/usr/local/bin/vroomy"); err != nil {
			out.Warningf("Unable to set cap on vroomy: %v", err)
			out.Notification("Note - you can grant vroomy permission to bind on reserved ports using setcap on linux:\n  `./vroomy/bin/setcap /usr/local/bin/vroomy` (linux)")
		} else {
			setcap = true
		}
	} else if lib.File.RunCmd("which", "codesign") == nil {
		if err = lib.File.RunCmd("sudo", "./bin/codesign", "vroomySigner", "~/go/bin/vroomy"); err == nil {
		} else if err = lib.File.RunCmd("sudo", "./bin/codesign", "Development", "~/go/bin/vroomy"); err == nil {
			setcap = true
		} else {
			out.Warningf("Unable to codesign vroomy: %v", err)
			out.Notification("Note - you can grant vroomy permission to bind on reserved with codesign:\n  `./vroomy/bin/codesign \"signing identity\" ~/go/bin/vroomy` (macosx - read CODESIGN.md for more info)")
		}

	} else {
		out.Notification("No codesigning or set cap available to grant permission for port binding.")
	}

	if setcap {
		out.Notification("Granted permission for port binding.")
	}

	if len(originalBranch) > 0 {
		lib.File.CheckoutBranch(originalBranch)
	}

	out.Successf("Installed vroomy %s successfully!", version)
	err = nil
	return
}
