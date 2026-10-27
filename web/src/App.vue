<template>
  <div class="min-h-screen bg-slate-50 text-slate-900">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
        <a
          href="/"
          class="text-xl font-bold tracking-tight text-emerald-700"
          aria-label="Pulso home"
        >
          Pulso
        </a>
        <span class="text-sm font-medium text-slate-500">Service monitoring</span>
      </div>
    </header>
    <main class="mx-auto max-w-7xl px-6 py-10">
      <RouterView />
    </main>
  </div>
</template>
