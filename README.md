This program generates Go-lang string constants or functions from given text.

The text literals are given in Markdown-based format which is described as follows.

# Build

Command to build the CLI:

```sh
go build github.com/yinyin/go-literal-code-gen
```

# Input Example

Each first level heading starts a new text literal with exception of heading **Heading Code** which will define top part of code file.

After heading, there are options and text content in bullets and fenced block.

Options will affect how texts are pre-processed before convert to string literal.

Text content is the desired literal text. An optional language parameters can be add to the fenced block to activate language specific processing.

````````markdown
# Heading Code

* `tail-new-line`
* `strip-spaces`

```go
package literal

import (
	"strconv"
)
```

# Literal Constant 1

* `const`: `literalOne`
* `strip-spaces`
* `replace`:
  - ```An ([a-z]+) a```
  - `$1`
  - ```banana```

```text
An apple a day
```
````````

# Options

Content lines will be process with the following order:

1. global options
2. language filter
3. replace rules


## Global Options

* `const`: `(CONSTANT_NAME)` - Generate constant.
* `builder`: `(FUNCTION_NAME)`, `(PARAMETER_DEFINITIONS)` - Generate builder function.
* `strip-spaces` - Remove prefix and suffix spaces.
* `preserve-new-line` - Generate new line character for all lines.
* `tail-new-line` - Generate tail new line character.
* `disable-language-filter` - Do not run language specific processing.

## Language Options

* SQL (`sql`):
    - `keep-comment`: do not strip comment lines

## Replace Rules

Replace rule is placed with global options in the following form:

``````markdown
* `replace`:
  - ```The `([regula]+)r` expression to trigger replacement ```
  - `$1`
  - ``` Substitute Code ```
``````
