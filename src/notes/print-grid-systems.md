---
layout: note.html
title: Grid Systems from Print to Web
date: 2024-02-20
tags:
 - note
 - design
 - css
published: true
excerpt: How print grid theory translates to the web.
image: /assets/print-grid.png
---

# Grid Systems from Print to Web

Print designers have used grids for centuries. The web is catching up.

## The Baseline Grid

A baseline grid aligns text to a repeating vertical unit. Every element's height and spacing is a **multiple of the base value**.

```css
:root {
    --base: 8px;
    --line-height: calc(var(--base) * 3);
}
```

## Why It Works

Consistent vertical rhythm makes a page feel calm and intentional, even when the reader can't name why.[^1]

## Definition List

Baseline grid
:   A grid of horizontal lines spaced at a fixed interval that text sits on

Vertical rhythm
:   The consistent spacing between lines and elements on a page

Leading
:   The space between lines of text, named after the lead strips used in typesetting

[^1]: Jan Tschichold's *The New Typography* (1928) is the canonical reference for this system.
