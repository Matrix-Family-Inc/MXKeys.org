/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { render } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import { HomePage } from './ui/home-page';

describe('HomePage', () => {
  it('renders the landing shell id', () => {
    const { container } = render(<HomePage />);
    // #home is the scroll-target hub every in-page nav link points at.
    expect(container.querySelector('#home')).not.toBeNull();
  });
});
