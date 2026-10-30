import { mount } from 'svelte';
import './app.css';
import App from './App.svelte';
import { connect } from './lib/net';
import { resumeAudioOnGesture } from './lib/audio.svelte';
import { initSkin } from './lib/skin.svelte';
import { initTooltips } from './lib/tooltip';

initSkin(); // Barock oder C64, vor dem ersten Zeichnen

// Verbindung zum Spielserver aufbauen und die Oberfläche einmalig einhängen.
connect();
resumeAudioOnGesture(); // Musik startet erst nach der ersten Nutzeraktion (Browser-Regel)
mount(App, { target: document.getElementById('app')! });
initTooltips(); // alle title-Hinweise im Spieldesign statt System-Tooltip
