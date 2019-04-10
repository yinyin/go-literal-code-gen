# Build

Command to build the CLI:

```sh
go build github.com/yinyin/go-literal-code-generator/cmd/go-literal-code-gen
```

# Example

````````markdown
# Heading Code

* `tail-new-line`
* `strip-spaces`

```
package literal

import (
	"strconv"
)

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
