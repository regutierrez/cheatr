# Python Corpus Samples

This file shows two **authoring examples** for Python concept pages described in `specs/cheatr-v1-scoped-search-tui.md`:

- a **full** example using all recommended canonical sections
- a **minimal valid** example using only a subset of sections

These are reference samples for humans and tools. They are **not** compiled directly unless copied into `content/python/*.md`.

## Sample A — Full card with all recommended sections

````md
---
id: python/list
scope: python
title: List
aliases: [lists, array]
keywords: [sequence, mutable, ordered]
version_verified: "Python 3.13"
docs_url: "https://devdocs.io/python~3.13/"
related: [python/tuple, python/set, python/dict]
---

# List

```python
items = []
items = [1, 2, 3]
items = list(iterable)
```

## Syntax

Python lists are mutable, ordered sequences.
Use `[]` for literals and `list(iterable)` when converting from another iterable.

## Create

```python
empty = []
nums = [1, 2, 3]
chars = list("abc")
```

## Access

```python
nums[0]
nums[-1]
nums[1:3]
```

## Modify

```python
nums = [1, 2]
nums.append(3)
nums.extend([4, 5])
nums[0] = 10
```

## Iterate

```python
nums = [1, 2, 3]
for n in nums:
    print(n)
```

## Notes

- Lists are mutable.
- Lists preserve insertion order.
- Lists can contain mixed types, though consistent types are usually clearer.

## Pitfalls

- `list()` and `[]` both work, but `[]` is usually the clearer literal syntax.
- `[[0] * 3] * 2` duplicates references to the same inner list.

## Related

- `tuple` for immutable ordered collections
- `set` for unique unordered elements
- `dict` for key/value mappings
````

## Sample B — Minimal valid card with only a subset of sections

````md
---
id: python/tuple
scope: python
title: Tuple
aliases: [tuples]
keywords: [sequence, immutable, ordered]
version_verified: "Python 3.13"
docs_url: "https://devdocs.io/python~3.13/"
related: [python/list]
---

# Tuple

```python
point = (1, 2)
single = (1,)
items = tuple(iterable)
```

## Create

```python
empty = ()
point = (1, 2)
letters = tuple("abc")
```

## Access

```python
point = (10, 20, 30)
point[0]
point[-1]
point[1:]
```

## Notes

Tuples are ordered like lists, but they are immutable.
Use a trailing comma for a one-item tuple.
````

## What these samples demonstrate

### Required in every Python concept page
- frontmatter with required fields
- one `# Title` heading
- one top syntax code block immediately after the title
- at least one `##` section

### Optional / subset-friendly
- you do **not** need every recommended section on every page
- `Syntax`, `Modify`, `Iterate`, `Pitfalls`, and `Related` may be omitted when they do not add enough value
- missing sections simply do not become searchable section labels for that page

### Indexed vs display-only headings
- `##` headings become canonical searchable sections
- `###` headings are display-only in v1
