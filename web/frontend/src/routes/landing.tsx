import { createFileRoute } from "@tanstack/react-router"

function LandingPage() {
  return (
    <div className="h-dvh w-full overflow-hidden bg-white">
      <iframe
        title="JameClaw landing page"
        src="/jameclaw-landing.html"
        className="block h-full w-full border-0"
      />
    </div>
  )
}

export const Route = createFileRoute("/landing")({
  component: LandingPage,
})
