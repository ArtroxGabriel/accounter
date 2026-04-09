# Design System: The Tactile Ledger

## 1. Overview & Creative North Star

**Creative North Star: "The Editorial Archive"**
Personal finance is often treated as a cold, mechanical task. This design system rejects that notion, instead treating financial data as a curated, high-end editorial experience. We are moving away from the "SaaS-dashboard" aesthetic and toward a "Soft Minimalist" aesthetic that feels as intentional as a physical leather-bound ledger.

The visual identity is defined by **intentional asymmetry, expansive breathing room, and tonal depth.** By leveraging a warm, cream-based palette and sophisticated layering, we create an environment of "Quiet Authority"—where the user feels in control, not overwhelmed. We break the grid through overlapping elements and high-contrast typography scales that guide the eye through narrative, not just data.

---

### 2. Colors & Tonal Architecture

The palette is rooted in a soft, organic cream to reduce eye strain and provide a premium "paper" feel.

#### Core Palette (Material Design Tokens)

* **Surface (Background):** `#fbf6ec` (The foundation of the experience)
* **Primary (Brand/Action):** `#006762` (Deep Teal for authority)
* **Secondary (Accents):** `#0c637d` (Quiet Blue)
* **Tertiary (Warmth):** `#95423c` (Editorial Clay)

#### The "No-Line" Rule

**Explicit Instruction:** Prohibit the use of 1px solid borders for sectioning. Structural boundaries must be defined solely through background color shifts.

* Use `surface-container-low` (`#f5f0e5`) to define a section against the `surface` background.
* Use `surface-container-highest` (`#e1dcd0`) for the most prominent interactive cards.

#### The Glass & Gradient Rule

To prevent a "flat" or "template" appearance:

* **Main CTAs:** Use a subtle linear gradient from `primary` (`#006762`) to `primary-dim` (`#005a55`) at a 145° angle.
* **Floating Elements:** Utilize **Glassmorphism**. Apply `surface-container-lowest` at 70% opacity with a `24px` backdrop blur to create a "frosted glass" effect for navigation bars or modals.

---

### 3. Typography: The Hierarchical Voice

We pair the structural precision of **Inter** with the editorial character of **Manrope** to create a system that feels both functional and artisanal.

| Level | Token | Font | Size | Character |
| :--- | :--- | :--- | :--- | :--- |
| **Display** | `display-lg` | Manrope | 3.5rem | Bold, tight tracking (-2%). Use for total balances. |
| **Headline** | `headline-md` | Manrope | 1.75rem | Medium. Used for section headers. |
| **Title** | `title-lg` | Inter | 1.375rem | Semi-bold. Used for card titles. |
| **Body** | `body-lg` | Inter | 1rem | Regular. The workhorse for all data. |
| **Label** | `label-md` | Inter | 0.75rem | Medium, All-Caps. Used for category tags. |

**The Editorial Scale:** Do not be afraid of the contrast between `display-lg` and `body-sm`. Large, confident numbers next to tiny, precise labels create a premium, "designed" feel.

---

### 4. Elevation & Depth: Tonal Layering

Traditional shadows and borders are replaced by the **Layering Principle.**

* **Nesting Depth:** Instead of shadows, stack surfaces. Place a `surface-container-lowest` (`#ffffff`) card on top of a `surface-container` (`#ede8dc`) background. The 4% difference in luminance is enough to signify elevation without visual clutter.
* **Ambient Shadows:** If a floating element (like a FAB) requires a shadow, use a "Tinted Ambient" approach:
  * `Box-shadow: 0 12px 32px -4px rgba(48, 47, 40, 0.08);`
  * Note: The shadow color is a low-opacity version of `on-surface`, never pure black.
* **The Ghost Border:** If accessibility requires a border, use `outline-variant` at **15% opacity**. It should be a suggestion of a line, not a hard stop.

---

### 5. Components

#### Buttons (The Soft-Touch Approach)

* **Primary:** Gradient fill (`primary` to `primary-dim`), `1rem` (xl) rounded corners. White `on-primary` text.
* **Secondary:** `surface-container-high` background with `primary` text. No border.
* **Iconography:** Always pair button text with a 20px optical size icon for immediate recognition.

#### Cards & Lists (The Borderless List)

* **Rule:** Forbid divider lines.
* **Implementation:** Use 16px of vertical whitespace to separate transactions.
* **Contextual Nesting:** For transaction groups (e.g., "Yesterday"), wrap the group in a `surface-container-low` wrapper with an `xl` (1.5rem) corner radius.

#### Input Fields

* **Style:** Minimalist "Underline" or "Soft Box."
* **Active State:** Transition the background to `surface-container-highest`. Use the `primary` color only for the cursor and the label-text to avoid "color-vomit."

#### Category Chips (The Signature Palette)

Use the diverse expense palette for Chips, but apply it with "Tonal Tinting":

* **Teal Chip:** Background: `#76C7C0` at 15% opacity. Text: `#006762`.
* **Coral Chip:** Background: `#F28B82` at 15% opacity. Text: `#95423c`.
* *This ensures the UI remains colorful but professional, preventing high-vibrancy "neon" clashes against the cream background.*

---

### 6. Do’s and Don’ts

**DO:**

* **DO** use white space as a structural element. If a design feels "messy," add 8px of padding before adding a line.
* **DO** use `surface-container-lowest` (#ffffff) for the most "active" or "clickable" cards to make them pop against the cream background.
* **DO** use Manrope for any numerical data you want to emphasize. It is our "hero" typeface.

**DON'T:**

* **DON'T** use pure black (#000000) for text. Use `on-surface` (#302f28) to maintain the soft, premium feel.
* **DON'T** use 1px dividers to separate list items. Use tonal shifts or generous spacing.
* **DON'T** use sharp 90-degree corners. Everything must feel "held"—use a minimum of `0.5rem` (md) for all interactive elements.

**Director's Note:** This system is about the *feel* of the data. Every tap should feel like turning a page in a high-end magazine. Keep it breathable, keep it tonal, and keep it intentional.
