/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { afterEach, describe, expect, it, vi } from 'vitest';
import { cleanup, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import { NotaryLookupForm } from './notary-lookup-form';

afterEach(cleanup);

describe('NotaryLookupForm', () => {
  it('accepts a valid server name and calls onSubmit', async () => {
    const onSubmit = vi.fn();
    render(<NotaryLookupForm onSubmit={onSubmit} />);

    const input = screen.getByLabelText(/matrix server/i);
    await userEvent.type(input, 'matrix.example.org');
    await userEvent.click(screen.getByRole('button', { name: /check notary keys/i }));

    expect(onSubmit).toHaveBeenCalledWith(
      { server_name: 'matrix.example.org' },
      expect.anything(),
    );
  });

  it('shows an accessible error for an invalid server name', async () => {
    const onSubmit = vi.fn();
    render(<NotaryLookupForm onSubmit={onSubmit} />);

    await userEvent.type(screen.getByLabelText(/matrix server/i), 'not valid!');
    await userEvent.click(screen.getByRole('button', { name: /check notary keys/i }));

    const alert = await screen.findByRole('alert');
    expect(alert).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it('disables the submit button when busy is true', () => {
    render(<NotaryLookupForm onSubmit={() => undefined} busy />);
    expect(screen.getByRole('button')).toBeDisabled();
  });
});
