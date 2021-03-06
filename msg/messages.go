package msg

var Welcome = `
Welcome to decentragit!

Your decentragit username has been created as {{.username | bold | yellow}}. Others can grant you access to their repos by running: {{print "git dg team add " .username | bold | cyan}}.
`

var AddDgitToRemote = `
decentragit would like to add {{.repourl | bold | yellow}} to the {{.remote | bold }} remote. This allows {{"git push" | bold | cyan}} to mirror this repository to decentragit.
`

var AddDgitToRemoteConfirm = `{{"Is that ok?" | bold | green}}`

var AddedDgitToRemote = `
Success, decentragit is now superpowering the {{.remote | bold }} remote.
Continue using your normal git workflow and enjoy being decentralized.
`

var AddDgitRemote = `
decentragit would like to add the {{.remote |  bold }} remote to this repo so that you can fetch directly from decentragit.
`

var AddDgitRemoteConfirm = AddDgitToRemoteConfirm

var AddedDgitRemote = `
Success, decentragit is now accessible under the {{.remote | bold }} remote.
{{print "git fetch " .remote | bold | cyan}} will work flawlessly from your decentralized repo.
`

var FinalInstructions = `
You are setup and ready to roll with decentragit.
Just use git as you usually would and enjoy a fully decentralized repo.

If you would like to clone this decentragit repo on another machine, simply run {{print "git clone " .repourl | bold | cyan}}.

If you use GitHub for this repo, we recommend adding a decentragit action to keep your post-PR branches in sync on decentragit.
You can find the necessary action here:
{{"https://github.com/quorumcontrol/dgit-github-action" | bold | blue}}

Finally for more docs or if you have any issues, please visit our github page:
{{"https://github.com/quorumcontrol/dgit" | bold | blue}}
`

var PromptRepoNameConfirm = `It appears your repo is {{.repo | bold | yellow}}, is that correct?`

var PromptRepoName = `{{"What is your full repo name?" | bold | green}}`

var PromptRepoNameInvalid = `Enter a valid repo name in the form '${user_or_org}/${repo_name}'`

var PromptRecoveryPhrase = `Please enter the recovery phrase for {{.username | bold | yellow}}: `

var PromptInvalidRecoveryPhrase = `Invalid recovery phrase, must be 24 words separated by spaces`

var IncorrectRecoveryPhrase = `
{{"Incorrect recovery phrase:" | bold | red}} the given phrase does not provide ownership for {{.username | bold | yellow}}. Please ensure recovery phrase and username is correct.
`

var PrivateKeyNotFound = `
Could not load your decentragit private key from {{.keyringProvider | bold }}. Try running {{"git dg init" | bold | cyan}} again.
`

var UserSeedPhraseCreated = `
Below is your recovery phrase, you will need this to access your account from another machine or recover your account.

{{"Please write this down in a secure location. This will be the only time the recovery phrase is displayed." | bold }}

{{.seed | bold | magenta}}
`

var UserNotFound = `
{{print "user " .user " does not exist" | bold | red}}
`

var UserNotConfigured = "\nNo decentragit username configured. Run `git config --global {{.configSection}}.username your-username`.\n"

var UserRestored = `
Your decentragit user {{.username | bold | yellow}} has been restored. This machine is now authorized to push to decentragit repos it owns.
`

var RepoCreated = `
Your decentragit repo has been created at {{.repo | bold | yellow}}.

decentragit repo identities and authorizations are secured by Tupelo - this repo's unique id is {{.did | bold | yellow}}.

Storage of the repo is backed by ScPrime Public Portals.
`

var RepoNotFound = `
{{"decentragit repository does not exist." | bold | red}}

You can create a decentragit repository by running {{"git dg init" | bold | cyan}}.
`

var RepoNotFoundInPath = `
{{print "No .git directory found in " .path | bold | red}}.

Please change directories to a git repo and run {{.cmd | bold | cyan}} again.

If you would like to create a new repo, use {{"git init" | bold | cyan}} normally and run {{.cmd | bold | cyan}} again.
`

var UsernamePrompt = `
{{ "What decentragit username would you like to use?" | bold | green }}
`
