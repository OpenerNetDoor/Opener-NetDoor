/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: "standalone",
  transpilePackages: ["@opener-netdoor/sdk-ts", "@opener-netdoor/shared-types"]
};

module.exports = nextConfig;
