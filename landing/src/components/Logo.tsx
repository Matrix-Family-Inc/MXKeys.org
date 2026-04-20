/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: 2026-01-28 UTC
 * Status: Created
 */

type LogoProps = {
  size?: number;
  animated?: boolean;
};

export function Logo({ size = 40, animated = false }: LogoProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 1024 1024"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={animated ? 'animate-spin-slow' : ''}
      style={animated ? { animationDuration: '20s' } : undefined}
      aria-hidden="true"
      focusable="false"
    >
      {/* Federation ring */}
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
      {/* Inner halo */}
      <circle cx="524" cy="500" r="96" fill="var(--color-bg-surface, #1c1c1e)" />
      {/* Core node */}
      <circle cx="524" cy="500" r="52" fill="var(--color-primary, #3D9970)" />
    </svg>
  );
}
