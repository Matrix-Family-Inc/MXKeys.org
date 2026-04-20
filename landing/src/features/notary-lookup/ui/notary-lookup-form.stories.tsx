/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import type { Meta, StoryObj } from '@storybook/react';

import { NotaryLookupForm } from './notary-lookup-form';

const meta: Meta<typeof NotaryLookupForm> = {
  title: 'Features / NotaryLookupForm',
  component: NotaryLookupForm,
  args: {
    // In Storybook we want to observe what the component would send
    // without a backend; console.info mirrors the "Actions" addon for
    // the barebones (no-addons) setup we currently run with.
    onSubmit: (input) => {
      console.info('[storybook] submit', input);
    },
  },
};
export default meta;

type Story = StoryObj<typeof NotaryLookupForm>;

export const Idle: Story = {};

export const Busy: Story = {
  args: { busy: true },
};
