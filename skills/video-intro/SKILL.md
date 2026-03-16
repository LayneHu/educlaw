---
name: video-intro
description: Find and embed an educational video for a topic, or generate a video script. Use when introducing a new concept, providing additional explanation through video, or when the student prefers video learning.
---

# Video Introduction

Find or recommend educational videos for a topic.

## When to use
- Introducing a brand new concept
- Student prefers video learning
- Topic benefits from demonstration (science experiments, etc.)
- Student is stuck and needs a different explanation style

## Process
1. Identify the topic and student grade level
2. Create an HTML card with video recommendations
3. Include Bilibili or YouTube search links for Chinese curriculum
4. Call render_content with type="video"

## Video Sources
- Bilibili: excellent for Chinese curriculum (人教版) content
- Khan Academy: for math and science
- Include direct search links

## Output
HTML card with:
- Recommended video titles
- Platform links (Bilibili/YouTube)
- Brief description of what to watch for
- Post-viewing reflection questions
