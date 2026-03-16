---
name: lesson-planner
description: Generate a structured lesson plan for teachers. Use when a teacher needs to prepare for class, create differentiated activities, or develop curriculum aligned with Chinese national standards (人教版). Outputs a complete lesson plan.
---

# Lesson Planner

Generate structured lesson plans aligned with Chinese national curriculum standards.

## When to use
- Teacher needs to prepare for a specific topic
- Creating differentiated activities for different ability levels
- Building a unit plan
- Preparing for inspections or evaluations

## Process
1. Read PROFILE.md to understand the teacher's subject and grade
2. Read CLASSES.md for class information
3. Generate a complete lesson plan
4. Call render_content with type="report" (lesson plan format)

## Lesson Plan Structure (人教版 format)
1. **基本信息** (Basic Info): Subject, Grade, Topic, Duration
2. **学习目标** (Learning Objectives): Knowledge, Skills, Values
3. **重难点** (Key Points & Difficulties)
4. **教学过程** (Teaching Process):
   - 导入 (Introduction): 5 min
   - 新课教学 (New Content): 20 min
   - 练习巩固 (Practice): 10 min
   - 总结 (Summary): 5 min
5. **板书设计** (Blackboard Design)
6. **作业布置** (Homework Assignment)
7. **差异化教学** (Differentiated Activities)

## Curriculum Alignment
- Reference 人教版 (PEP) textbook standards
- Align with 新课程标准 (New Curriculum Standards)
- Include cross-subject connections where appropriate
