# Contribute to the KubeEdge Website
Thanks for taking the time to join our community and start contributing!

## Quick guide to working with a GitHub repo

Here's a quick guide to a fairly standard GitHub workflow. This section is handy
for people who don't use git or GitHub often, and just need a quick guide to
get going:

1. Fork the kubeedge/website repo:

    * Go to the [kubeedge/website repo][kubeedge-website-repo] on GitHub.
    * Click **Fork** to make your own copy of the repo. GitHub creates a copy
      at `https://github.com/<your-github-username>/website`.

2. Open a command window on your local machine.

3. Clone your forked repo, to copy the files down to your local machine.
  This example creates a directory called `kubeedge` and uses SSH cloning to
  download the files:

    ```
    mkdir kubeedge
    cd kubeedge/
    git clone git@github.com:<your-github-username>/kubeedge-website.git
    cd kubeedge-website/
    ```

4. Add the upstream repo as a git remote repo:

    ```
    git remote add upstream https://github.com/kubeedge/website.git
    git remote set-url --push upstream no_push
    ```

5. Check your remotes:

    ```
    git remote -v
    ```

    You should have 2 remote repos:

      -  `origin` - points to your own fork of the repo on gitHub -
         that is, the one you cloned my local repo from.
      -  `upstream` - points to the actual repo on gitHub.

6. Create a branch. In this example, replace `doc-updates` with any branch name
  you like. Choose a branch name that helps you recognise the updates you plan
  to make in that branch:

    ```
    git checkout -b doc-updates
    ```

7. Ensure you are in your target branch.

    ```
    git branch
    ```

    The branch mark with `*` is your branch now.

8. Add and edit the files as you like. The doc pages are in the
  `/<LANG-CODE>/content/docs/` directory.  

**You can use the guide [here](https://sourcethemes.com/academic/docs/writing-markdown-latex/) for formatting your content and using shortcodes.**  

9. Run `git status` at any time, to check the status of your local files.
  Git tells you which files need adding or committing to your local repo.

10. Commit your updated files to your local git repo. Example commit:

    ```
    git commit -a -m "Fixed some doc errors."
    ```

    Or:

    ```
    git add add-this-doc.md
    git commit -a -m "Added a shiny new doc."
    ```

11. Push from your branch (for example, `doc-updates`) to **the relevant branch
  on your fork on GitHub:**

    ```
    git checkout doc-updates
    git push origin doc-updates
    ```

12. When you're ready to start the review process, create a pull request (PR)
  **in the branch** on **your fork** on the GitHub UI, based on the above push.
  The PR is auto-sent to the upstream repo - that is, the one you forked from.

13. If you need to make changes to the files in your PR, continue making them
  locally in the same branch, then push them again in the same way. GitHub
  automatically sends them through to the same PR on the upstream repo!

14. **Hint:** If you're authenticating to GitHub via SSH, use `ssh-add` to add
  your SSH key passphrase to the managing agent, so that you don't have to
  keep authenticating to GitHub. You need to do this again after every reboot.

Please remember read and observe the [Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

[kubeedge-website-repo]: https://github.com/kubeedge/website