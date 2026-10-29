import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Cache Components is on in the create-next-app 16 empty template. Leave it
  // off until a later feature needs explicit caching; placeholder pages should
  // render without "use cache" / Suspense ceremony.
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
