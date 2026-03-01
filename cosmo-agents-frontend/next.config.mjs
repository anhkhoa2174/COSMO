import bundleAnalyzer from '@next/bundle-analyzer';
import { withSentryConfig } from '@sentry/nextjs';

const withBundleAnalyzer = bundleAnalyzer({
  enabled: process.env.ANALYZE === 'true',
});

export default withSentryConfig(
  withBundleAnalyzer({
    output: 'standalone',
    reactStrictMode: false,
    eslint: {
      ignoreDuringBuilds: true,
    },
    experimental: {
      optimizePackageImports: ['@mantine/core'],
    },
    images: {
      remotePatterns: [
        {
          protocol: 'https',
          hostname: '**.cloudfront.net',
        },
        {
          protocol: 'https',
          hostname: '**.cloudinary.com',
        },
        {
          protocol: 'https',
          hostname: '**.amazonaws.com',
        },
      ],
    },
    webpack(config) {
      config.externals.push({ canvas: 'commonjs canvas' });

      const fileLoaderRule = config.module.rules.find((rule) =>
        rule.test?.test?.('.svg')
      );

      config.module.rules.push(
        {
          ...fileLoaderRule,
          test: /\.svg$/i,
          resourceQuery: /url/,
        },
        {
          test: /\.svg$/i,
          issuer: fileLoaderRule.issuer,
          resourceQuery: {
            not: [...fileLoaderRule.resourceQuery.not, /url/],
          },
          use: [{ loader: '@svgr/webpack', options: { icon: 24 } }],
        }
      );

      fileLoaderRule.exclude = /\.svg$/i;

      return config;
    },
  }),
  {
    org: 'rockship-06',
    project: 'cosmo-agents-fe',
    silent: true,
    widenClientFileUpload: false,
    transpileClientSDK: false,
    tunnelRoute: '/monitoring',
    hideSourceMaps: true,
    disableLogger: true,
    automaticVercelMonitors: false,
    sourcemaps: {
      disable: true,
      deleteSourcemapsAfterUpload: true,
    },
  }
);
