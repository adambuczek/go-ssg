---
layout: note.html
title: Understanding Rust Ownership
date: 2024-01-10
tags:
 - note
 - rust
 - systems
published: true
excerpt: A brief look at Rust ownership and why it matters.
image: /assets/rust-ownership.png
---

# Understanding Rust Ownership

Rust's ownership system is its most distinctive feature. It eliminates entire classes of bugs at compile time with **no runtime cost**.

## The Rules

Every value has exactly one owner. When the owner goes out of scope, the value is dropped.

```rust
fn main() {
    let s = String::from("hello");
    takes_ownership(s);
    // s is no longer valid here
}
```

## Borrowing

You can lend a value without transferring ownership.[^1]

```rust
fn length(s: &String) -> usize {
    s.len()
}
```

## Definition List

Owner
:   The variable that holds a value and is responsible for freeing it

Borrow
:   A reference to a value that does not take ownership

Lifetime
:   A scope annotation telling the compiler how long a reference is valid

[^1]: Borrowing rules are checked at compile time by the borrow checker.
