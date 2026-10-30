<script lang="ts">
  // Wurzel: zeigt Lobby bis zum Spielstart, danach den Hauptbildschirm.
  import { app } from './lib/state.svelte';
  import Lobby from './components/Lobby.svelte';
  import Game from './components/Game.svelte';
  import Toasts from './components/Toasts.svelte';
  import Beitrittsanfragen from './components/Beitrittsanfragen.svelte';
</script>

{#if app.started && app.self}
  <Game />
{:else}
  <Lobby />
{/if}
<Beitrittsanfragen />
<Toasts />
{#if app.connection !== 'open'}
  <div class="offline" role="status">Verbindung zum Server wird hergestellt …</div>
{/if}

<style>
  .offline {
    position: fixed; left: 50%; bottom: 16px; transform: translateX(-50%);
    background: var(--ink); color: var(--gold-light); padding: 10px 18px; border-radius: 6px;
    font-family: var(--font-label); font-size: 13px; letter-spacing: .06em;
  }
</style>
