# Extended Markdown Features

mdPress extends standard Markdown with powerful features for technical documentation, including mathematical equations, diagrams, footnotes, and auto-linking. This guide covers the advanced Markdown capabilities available in mdPress.

## KaTeX Mathematical Expressions

Write mathematical equations using KaTeX syntax. Both inline and display modes are supported.

### Inline Math

Use single dollar signs to write inline mathematical expressions:

```markdown
The quadratic formula is $ax^2 + bx + c = 0$.

Einstein's famous equation is $E = mc^2$.

The sum of the first $n$ natural numbers is $\frac{n(n+1)}{2}$.
```

### Display Math

Use double dollar signs for display equations that render on their own line:

```markdown
The Pythagorean theorem:

$$a^2 + b^2 = c^2$$

The formula for a circle:

$$A = \pi r^2$$
```

### Complex Equations

mdPress supports all standard KaTeX functions and symbols:

```markdown
$$\int_0^\infty e^{-x^2} dx = \frac{\sqrt{\pi}}{2}$$

$$\frac{\partial f}{\partial x} = \lim_{h \to 0} \frac{f(x+h) - f(x)}{h}$$

$$\begin{pmatrix} a & b \\ c & d \end{pmatrix} \begin{pmatrix} x \\ y \end{pmatrix} = \begin{pmatrix} ax + by \\ cx + dy \end{pmatrix}$$
```

### Best Practices for Math

- Use display mode (`$$`) for important equations
- Use inline mode (`$`) for brief expressions within text
- Add explanatory text before complex equations
- Test rendering in the live preview to ensure correct output

## Mermaid Diagrams

Create flowcharts, sequence diagrams, state diagrams, and more using Mermaid syntax.

### Flowcharts

Visualize processes and decision trees:

```markdown
​```mermaid
flowchart TD
    A[Start] --> B{Decision}
    B -->|Yes| C[Process A]
    B -->|No| D[Process B]
    C --> E[End]
    D --> E
​```
```

### Sequence Diagrams

Show interactions between systems or actors:

```markdown
​```mermaid
sequenceDiagram
    actor User
    User->>Browser: Click button
    Browser->>Server: Send request
    Server->>Database: Query data
    Database-->>Server: Return results
    Server-->>Browser: Send response
    Browser-->>User: Display result
​```
```

### State Diagrams

Document state machines and workflows:

```markdown
​```mermaid
stateDiagram-v2
    [*] --> Idle
    Idle --> Loading: User clicks
    Loading --> Ready: Data loaded
    Ready --> Processing: Start button
    Processing --> Done: Complete
    Done --> [*]
​```
```

### Class Diagrams

Model object-oriented systems:

```markdown
​```mermaid
classDiagram
    class Animal {
        -String name
        +String getName()
        +void setName(String)
    }
    class Dog {
        +void bark()
    }
    Animal <|-- Dog
​```
```

### Gantt Charts

Plan projects with timeline visualization:

```markdown
​```mermaid
gantt
    title Project Timeline
    section Design
    Wireframes :d1, 2026-03-01, 10d
    Mockups :d2, after d1, 10d
    section Development
    Frontend :d3, 2026-03-15, 20d
    Backend :d4, 2026-03-20, 25d
    section Testing
    QA :d5, after d3, 10d
​```
```

### Pie Charts

Display data distribution:

```markdown
​```mermaid
pie title Browser Market Share
    "Chrome" : 65
    "Firefox" : 15
    "Safari" : 15
    "Others" : 5
​```
```

### Supported Diagram Types

Mermaid diagrams in mdPress support: flowchart, sequence, state, class, entity-relationship, git graph, user journey, and pie charts. Additional diagram types may be available depending on your mdPress version.

## PlantUML Diagrams

Create complex diagrams using PlantUML syntax as an alternative to Mermaid:

```markdown
​```plantuml
@startuml
actor User
User -> WebApp: Click button
WebApp -> Server: Send request
Server -> Database: Query
Database --> Server: Return data
Server --> WebApp: Response
WebApp --> User: Display
@enduml
​```
```

### PlantUML Diagram Types

Supported diagram types include: use case, sequence, class, state, activity, component, deployment, object, and timing diagrams.

### When to Use PlantUML vs Mermaid

Use **Mermaid** for:
- Quick flowcharts and decision trees
- Simple sequence diagrams
- Gantt project timelines
- Data visualization

Use **PlantUML** for:
- Complex UML diagrams
- Detailed class hierarchies
- Component and deployment diagrams
- When you need precise control

## Footnotes

Add footnotes to provide additional context without interrupting the main text:

```markdown
Here is some text with a footnote[^1].

Another sentence with a different footnote[^2].

[^1]: This is the first footnote content.
[^2]: This is the second footnote. It can span
    multiple lines if you indent continuation lines.
```

### Footnote Features

- Footnotes are automatically numbered
- Readers can click footnote references to jump to the footnote
- Footnotes appear at the end of the document
- Use meaningful labels for complex documents with many footnotes

### Complex Footnote Content

Footnotes can contain Markdown formatting:

```markdown
The algorithm[^algo] is well-documented.

[^algo]: The **quicksort** algorithm:

    1. Select a pivot
    2. Partition array
    3. Recursively sort partitions
```

## Custom Heading IDs

While mdPress auto-generates heading IDs, you can specify custom IDs for special cases:

```markdown
# My Heading {#custom-id}

Some content here.

[Jump back](#custom-id)
```

Custom IDs are useful when:
- You change heading text but want stable links
- You need specific URL structure
- You're migrating from another documentation system
- You want short, memorable IDs

## Glossary Terms Auto-Linking

mdPress can automatically convert glossary terms to links. Define your glossary in the configuration:

```yaml
# book.yaml
glossary:
  - term: "API"
    definition: "Application Programming Interface"
    link: "./glossary.md#api"
  - term: "REST"
    definition: "Representational State Transfer"
    link: "./glossary.md#rest"
```

When `glossary` is configured, the first occurrence of each term is automatically linked. This appears in the text as highlighted terms that link to the glossary.

### Creating a Glossary Chapter

Create a glossary chapter with custom heading IDs:

```markdown
# Glossary

## API {#api}

Application Programming Interface - a set of protocols and tools for building software applications.

## REST {#rest}

Representational State Transfer - an architectural style for designing networked applications.

## JSON {#json}

JavaScript Object Notation - a lightweight data interchange format.
```

## Cross-References

Link between documents using relative paths and fragment identifiers:

```markdown
[See the installation guide](./installation.md)
[Review the API reference](./api-reference.md#authentication)
[Jump to advanced options](#advanced-options)
```

### Cross-Reference Best Practices

- Use relative paths starting with `./` or `../`
- Include fragment identifiers when linking to sections
- Test links in the live preview to ensure they work
- Avoid broken links in your final build

### Programmatic References

In complex documentation, you might reference sections programmatically:

```markdown
See [Section {{section-number}}](./reference.md) for details.
```

(Note: This example shows the concept; actual implementation depends on your mdPress configuration.)

## Advanced Features

### Combined Features

You can combine multiple extensions in sophisticated documentation:

```markdown
## Algorithm Analysis {#algorithm-analysis}

The time complexity is $O(n \log n)$. See the flowchart[^1] below:

​```mermaid
flowchart LR
    A[Input] --> B[Sort]
    B --> C[Output]
​```

[^1]: This diagram uses the merge sort approach.
```

### Escaping Special Characters

When you need literal dollar signs or other special characters:

```markdown
The price is \$50, not $50 as a formula.

Escape backticks: \`literal backticks\`
```

## Troubleshooting

### Math Not Rendering

Ensure dollar signs are not escaped and the KaTeX expression is valid. Test simple expressions first:

```markdown
Test: $x + y = z$
```

### Diagrams Not Displaying

- Verify the code block language identifier is correct (`mermaid` or `plantuml`)
- Check the syntax against official documentation
- Test with simpler diagrams first
- Check browser console for error messages

### Footnote Issues

- Ensure footnote labels match between reference and definition
- Footnote definitions can go anywhere after the reference
- Keep label names concise and meaningful

### Custom IDs Conflicting

- IDs must be unique within a document
- Avoid spaces and special characters in custom IDs
- Use lowercase letters, numbers, and hyphens only
