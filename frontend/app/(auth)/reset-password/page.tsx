import { Suspense } from "react";

import { ResetPasswordForm } from "@/components/ResetPasswordForm";
import { resetPasswordHeading } from "@/lib/copy";

export default function Page() {
  return (
    <Suspense
      fallback={
        <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-4 px-6 py-16">
          <p className="text-sm text-muted">{resetPasswordHeading}</p>
        </main>
      }
    >
      <ResetPasswordForm />
    </Suspense>
  );
}
