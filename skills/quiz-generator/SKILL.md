---
name: quiz-generator
description: Generate an interactive quiz or assessment for a student. Use when testing knowledge retention, reviewing before an exam, or doing formative assessment. Creates an engaging HTML quiz with immediate feedback.
---

# Quiz Generator

Generate an interactive HTML quiz for assessment or review.

## When to use
- Before or after teaching a topic to assess understanding
- Student wants to test themselves
- Review session before an exam
- Formative assessment

## Process
1. Read the student's KNOWLEDGE.md to understand their level
2. Use read_skill with asset_file="assets/quiz-template.html" for the template
3. Generate 5-10 questions appropriate to the level
4. Call render_content with type="quiz"

## Question Types
- Multiple choice (4 options)
- True/False
- Fill in the blank

## Design Principles
- Questions should progress from easy to hard
- Include brief explanations for wrong answers
- Show final score with encouraging message
