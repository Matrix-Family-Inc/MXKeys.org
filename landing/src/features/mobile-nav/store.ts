/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { create } from 'zustand';

type MobileNavState = {
  open: boolean;
  toggle: () => void;
  close: () => void;
};

/**
 * useMobileNav centralizes the mobile navigation open/closed state so the
 * header toggle and any nested link can cooperate without prop drilling or
 * context wiring. Kept deliberately tiny: a single boolean + two mutators.
 */
export const useMobileNav = create<MobileNavState>((set) => ({
  open: false,
  toggle: () => set((state) => ({ open: !state.open })),
  close: () => set({ open: false }),
}));
