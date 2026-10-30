// Dev-Werkzeug: zeigt Landeswappen und alle Wappenbilder groß, zum Prüfen der Zeichnungen.
import { mount } from 'svelte';
import '../app.css';
import Galerie from './Galerie.svelte';

mount(Galerie, { target: document.getElementById('app')! });
