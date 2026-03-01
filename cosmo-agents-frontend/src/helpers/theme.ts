'use client';

import {
  Anchor,
  Badge,
  Checkbox,
  createTheme,
  Drawer,
  Group,
  LoadingOverlay,
  NavLink,
  Radio,
  rem,
  ScrollArea,
  Stack,
  TagsInput,
  Text,
  TextInput,
} from '@mantine/core';

import classes from '@/styles/Default.module.css';

const theme = createTheme({
  black: 'var(--mantine-color-dark-6)',
  colors: {
    blackPearl: [
      '##e6e9e8',
      '#c0c7c7',
      '#9ba6a5',
      '#758483',
      '#4f6361',
      '#2a413f',
      '#04201d',
      '#031b19',
      '#031614',
      '#021210',
    ],
  },
  primaryColor: 'blackPearl',
  fontFamily: 'var(--font-primary)',
  headings: {
    fontFamily: 'var(--font-primary)',
  },
  radius: {
    xs: rem(4),
    sm: rem(6),
    md: rem(10),
  },
  defaultRadius: 'md',
  spacing: {
    xs: rem(8),
    lg: rem(24),
  },
  components: {
    Anchor: Anchor.extend({
      classNames: {
        root: classes.anchorRoot,
      },
    }),
    Badge: Badge.extend({
      classNames: {
        root: classes.badgeRoot,
      },
    }),
    Checkbox: Checkbox.extend({
      defaultProps: {
        radius: 'sm',
      },
    }),
    Drawer: Drawer.extend({
      defaultProps: {
        position: 'right',
      },
    }),
    Group: Group.extend({
      defaultProps: {
        gap: 'sm',
      },
    }),
    Radio: Radio.extend({
      classNames: {
        root: classes.radioIcon,
      },
    }),
    Stack: Stack.extend({
      defaultProps: {
        gap: 'sm',
      },
    }),
    ScrollArea: ScrollArea.extend({
      defaultProps: {
        scrollbarSize: 8,
        scrollHideDelay: 0,
      },
    }),
    Text: Text.extend({
      classNames: {
        root: classes.textRoot,
      },
    }),
    NavLink: NavLink.extend({
      styles: (_theme) => ({
        root: {
          borderRadius: _theme.radius.md,
          transition: 'background-color 0.3s ease',
        },
      }),
    }),
    TextInput: TextInput.extend({
      defaultProps: {
        labelProps: {
          style: {
            marginBottom: rem(4),
            // fontSize: rem(16),
          },
        },
      },
    }),
    LoadingOverlay: LoadingOverlay.extend({
      defaultProps: {
        loaderProps: {
          type: 'bars',
        },
      },
    }),
    TagsInput: TagsInput.extend({
      styles: (_theme) => ({
        pill: {
          backgroundColor: _theme.colors.blue[1],
        },
      }),
    }),
  },
});

export default theme;
