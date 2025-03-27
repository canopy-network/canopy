# Contributing Guidelines

*Pull requests, bug reports, and all other forms of contribution are welcomed and highly encouraged!* :octocat:

*Remember you can always find us on our [Discord server](https://discord.com/invite/pNcSJj7Wdh).*

## :inbox_tray: Opening an Issue

Before
[creating an issue](https://help.github.com/en/github/managing-your-work-on-github/creating-an-issue),
check if you are using the latest version of the project. If you are not up-to-date, see if updating
fixes your issue first.

### :beetle: Bug Reports and Other Issues

A great way to contribute to the project is to send a detailed issue when you encounter a problem.
We always appreciate a well-written, thorough bug report. :v:

In short, since you are most likely a developer, **provide a ticket that you would like to
receive**.

- **Review the [documentation](TODO) before opening a new issue.

- **Do not open a duplicate issue.** Search through existing issues to see if your issue has
  previously been reported. If your issue exists, comment with any additional information you have.
  You may simply note "I have this problem too", which helps prioritize the most common problems and
  requests.

- **Prefer using
  [reactions](https://github.blog/2016-03-10-add-reactions-to-pull-requests-issues-and-comments/)**,
  not comments, if you simply want to "+1" an existing issue.

- **Fully complete the provided issue template.** The bug report template requests all the
  information we need to quickly and efficiently address your issue. Be clear, concise, and
  descriptive. Provide as much information as you can, including steps to reproduce, stack traces,
  compiler errors, library versions, OS versions, and screenshots (if applicable).

- **Use
  [GitHub-flavored Markdown](https://help.github.com/en/github/writing-on-github/basic-writing-and-formatting-syntax).**
  Especially put code blocks and console outputs in backticks (```). This improves readability.

### :lock: Reporting Security Issues

**Do not** file a public issue for security vulnerabilities, message the maintainers privately
first.

## :repeat: Submitting Pull Requests

We **love** pull requests! Before
[forking the repo](https://help.github.com/en/github/getting-started-with-github/fork-a-repo) and
[creating a pull request](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/proposing-changes-to-your-work-with-pull-requests)
for non-trivial changes, it is usually best to first open an issue to discuss the changes, or
discuss your intended approach for solving the problem in the comments for an existing issue.

*Note: All contributions will be licensed under the project's license.*

- **Smaller is better.** Submit **one** pull request per bug fix or feature. A pull request should
  contain isolated changes pertaining to a single bug fix or feature implementation. **Do not**
  refactor or reformat code that is unrelated to your change. It is better to **submit many small
  pull requests** rather than a single large one. Enormous pull requests will take enormous amounts
  of time to review, or may be rejected altogether.

- **Coordinate bigger changes.** For large and non-trivial changes, open an issue to discuss a
  strategy with the maintainers. Or better yet, contact us directly on our
  [Discord](https://discord.gg/your-discord-link). Otherwise, you risk doing a lot of work for
  nothing!

- **Prioritize understanding over cleverness.** Write code clearly and concisely. Remember that
  source code usually gets written once and read often. Ensure the code is clear to the reader. The
  purpose and logic should be obvious to a reasonably skilled developer, otherwise you should add a
  comment that explains it.

- **Follow existing coding style and conventions.** Keep your code consistent with the style,
  formatting, and conventions in the rest of the code base. When possible, these will be enforced
  with a linter. Consistency makes it easier to review and modify in the future.

- **Include test coverage.** Add unit tests or UI tests when possible. Follow existing patterns for
  implementing tests.

- **Update the example project** if one exists to exercise any new functionality you have added.

- **Add documentation.** Document your changes with code doc comments or in existing guides.

- **Update the CHANGELOG** for all enhancements and bug fixes. Include the corresponding date, issue
  number if one exists and current version if any. Check the format of the [CHANGELOG](.docs/CHANGELOG.md).

- **Use the repo's default main branch.** Branch from and
  [submit your pull request](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/creating-a-pull-request-from-a-fork)
  to the repo's default branch `main`.

-
  **[Resolve any merge conflicts](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/resolving-a-merge-conflict-on-github)**
  that occur.

- **Promptly address any CI failures**. If your pull request fails to build or pass tests, please
  push another commit to fix it.

- When writing comments, use properly constructed sentences, including punctuation and **always** aim
  to add at least one comment per line of code, no matter how simple the statement is as we're focused
  on allowing the code to be easily understood by the most amount of developers possible.

### :nail_care: Coding Style

Consistency is the most important. Following the existing style, formatting, and naming conventions
of the file you are modifying and of the overall project. Failure to do so will result in a
prolonged review process that has to focus on updating the superficial aspects of your code, rather
than improving its functionality and performance.

Key requirements:

- **Use gofmt**: All Go code must be formatted using the standard `gofmt` tool. This ensures
  consistent formatting across the entire codebase. Run `gofmt -w .` before submitting your code.

- **Style Guide**: Follow the [Google Go Style Guide](https://google.github.io/styleguide/go/) for
  official recommendations on code structure and formatting. Additionally, review the project's
  existing code patterns to ensure your contributions maintain consistency.

- **Documentation**: Follow the official Go commentary guidelines:
  - Every exported type, function, and variable must have a doc comment
  - Comments should be complete sentences, starting with the item's name
  - Example: `// Transaction represents a single blockchain transaction.`

- **Branch Strategy**: All pull requests should be:
  - Based on the `development` branch
  - Opened against the `development` branch
  - Merged into `development` first before being promoted to `main`

- **EditorConfig Support**: We recommend using EditorConfig to maintain consistent coding styles.
  The project includes an `.editorconfig` file that defines common formatting rules.

- **Frontend Formatting**: Our Wallet and Explorer projects use `prettier` for consistent code
  formatting. Either run `npm run prettier` before submitting your PR or configure your editor to
  format on save using the project's `.prettierrc` settings.

If in doubt about any styling decisions, feel free to ask in the project's [Discord server](https://discord.com/invite/pNcSJj7Wdh).

### ✍️ Writing Package README.md Files

Each high-level package in the project must include a `README.md file` that explains its purpose,
functionality, and usage. This ensures that contributors and users can easily understand the role of
each package in the project. Follow these markdown guidelines to write effective and consistent
`README.md` files which are loosely based on the
[Microsoft's markdown best practices](https://learn.microsoft.com/en-us/powershell/scripting/community/contributing/general-markdown):

#### Blank Lines and Spacing

- Insert a single blank line between different Markdown blocks (e.g., between a paragraph and a list or header).
- Avoid multiple consecutive blank lines; they render as a single blank line in HTML.
- Within code blocks, consecutive blank lines can break the block.
- Remove trailing spaces at the end of lines, as they can affect rendering.
- Use spaces instead of tabs for indentation.

#### Titles and Headings

- Utilize ATX-style headings (`#` for H1, `##` for H2, etc.).
- Apply sentence case: capitalize only the first letter and proper nouns.
- Ensure a single space exists between the `#` and the heading text.
- Surround headings with a single blank line.
- Limit documents to one H1 heading.
- Increment heading levels sequentially without skipping (e.g., H2 should follow H1).
- Restrict heading depth to H3 or H4.
- Avoid using bold or other markup within headings.

#### Line Length

- Limit lines to 100 characters for conceptual articles and cmdlet references.
- For `about_` topics, restrict line length to 79 characters.
- This practice enhances readability and simplifies version control diffs.

#### Emphasis

- Use `**` for bold text.
- Use `_` for italicized text.
- Consistent use clarifies intent, especially when mixing bold and italics.

#### Fenced Code Blocks

- Use triple backticks (```) to denote code blocks.
- Specify the language immediately after the opening backticks for syntax highlighting.

Example:

```go
func main() {
  fmt.Println("hello world")
}
```

#### Image Guidelines

- Provide descriptive alt text for accessibility.
- Ensure images are relevant and enhance the content.
