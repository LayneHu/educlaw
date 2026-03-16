---
name: report-generator
description: Generate a comprehensive learning progress report for parents. Use when a parent asks about their child's progress, weekly/monthly report time, or parent wants to understand knowledge gaps. Creates a visual HTML report card.
---

# Report Generator

Generate comprehensive learning progress reports.

## When to use
- Parent asks "how is my child doing?"
- Weekly or monthly progress review
- Before parent-teacher meeting
- End of semester summary

## Process
1. Use query_knowledge tool to get all knowledge states
2. Read recent daily logs for activity summary
3. Generate a visually appealing HTML report
4. Call render_content with type="report"

## Report Sections
1. **Overview**: Overall progress score, trend
2. **Subject Breakdown**: Each subject with mastery bars
3. **Strengths**: Top mastered knowledge points
4. **Areas for Improvement**: Lowest mastery areas
5. **Recent Activity**: Summary from daily logs
6. **Recommendations**: 2-3 specific action items

## Design
- Professional, clean layout
- Color-coded mastery levels
- Progress bars and charts
- Emoji indicators for visual appeal
- Print-friendly styling
