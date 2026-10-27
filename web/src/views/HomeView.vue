<template>
  <section
    aria-labelledby="overview-title"
    class="rounded-xl border border-slate-200 bg-white p-8 shadow-sm"
  >
    <p class="mb-2 text-sm font-semibold uppercase tracking-wide text-emerald-700">
      Service monitoring
    </p>
    <h1 id="overview-title" class="text-3xl font-bold tracking-tight">
      Know when your services go down.
    </h1>
    <p class="mt-3 max-w-2xl text-slate-600">
      Pulso monitors availability, measures latency, and communicates service incidents.
    </p>
    <div class="mt-6 flex gap-3">
      <RouterLink
        to="/register"
        class="rounded-md bg-emerald-700 px-4 py-2 font-medium text-white hover:bg-emerald-800"
      >
        Create an account
      </RouterLink>
      <RouterLink
        to="/login"
        class="rounded-md border border-slate-300 px-4 py-2 font-medium text-slate-700"
      >
        Log in
      </RouterLink>
    </div>
  </section>
</template>
