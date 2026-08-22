/**
 * Design-system barrel — the operating system's furniture.
 *
 * Every screen in the product is composed from these: a `Panel` to hold a
 * region, a `StateLamp` to say what something is doing, a `Gauge` to draw a
 * count, a `Field` for one labelled reading, and a `ScreenTitle` at the head of
 * an application. The shell's own three bands (`StatusRail`, `CommandLine`,
 * `FunctionRail`) are exported too, though only the root layout mounts them.
 *
 * Interaction primitives — `Button`, `MenuItem` — live in `~/components/ui`.
 */
export { CommandLine } from "./command-line/command-line";
export { Field } from "./field/field";
export { FunctionRail } from "./function-rail/function-rail";
export { Gauge, litSegments } from "./gauge/gauge";
export { Panel } from "./panel/panel";
export { RegisterCell, lampToneFor } from "./register-cell/register-cell";
export { ScreenTitle } from "./screen/screen";
export { StateLamp } from "./lamp/lamp";
export { StatusRail } from "./status-rail/status-rail";
