/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Updated
 */

type LogoProps = {
  size?: number;
  className?: string;
  animated?: boolean;
};

/**
 * Canonical MXKeys brand mark: dashed federation ring with a
 * halo-bordered core node. Kept as a shared primitive so operators
 * forking the landing for their own branded notary can swap this
 * file without touching widgets. Colours route through
 * `--color-primary` / `--color-bg-surface` so the mark follows the
 * active theme without hard-coding hex values; the literal hex
 * fallbacks preserve the original design when the mark is rendered
 * outside the landing theme context (e.g. in Storybook stories or
 * embedded previews).
 */
export function Logo({ size = 32, className, animated = false }: LogoProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 1024 1024"
      width={size}
      height={size}
      fill="none"
      role="img"
      aria-label="MXKeys logo"
      className={[animated ? 'animate-spin-slow' : '', className].filter(Boolean).join(' ')}
      style={animated ? { animationDuration: '20s' } : undefined}
    >
      {/* Federation ring: broken arcs evoke federated servers. */}
      <circle
        cx="512"
        cy="512"
        r="360"
        stroke="var(--color-primary, #3D9970)"
        strokeWidth="76"
        strokeLinecap="round"
        strokeDasharray="260 320 200 360 220 420"
        transform="rotate(-18 512 512)"
        fill="none"
      />
      {/* Inner halo: creates the negative-space eye around the node. */}
      <circle cx="524" cy="500" r="96" fill="var(--color-bg-surface, #1c1c1e)" />
      {/* Core node: the notary itself. */}
      <circle cx="524" cy="500" r="52" fill="var(--color-primary, #3D9970)" />
    </svg>
  );
}
