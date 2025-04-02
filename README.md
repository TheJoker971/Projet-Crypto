# Kraken Archival

## Description

Ceci est un projet qui sert à me familiariser avec le language GO.
Celui-ci permet de récupérer des données sur les pairs de Kraken et de faire un système d'archivage.

## 📡 Routes API

### ✅ Données Kraken

| Méthode | Endpoint                  | Description                                                           |
|---------|---------------------------|-----------------------------------------------------------------------|
| `GET`   | `/status`                | Retourne le statut de l'API Kraken (online, maintenance, etc.)       |
| `GET`   | `/pairs`                 | Retourne la liste des paires archivées (via les fichiers CSV)        |
| `GET`   | `/pair/:namePair`        | Retourne les données actuelles de la paire depuis la base SQLite     |
| `GET`   | `/download/:namePair`    | Télécharge le dernier fichier CSV archivé pour la paire              |

---

### 📘 Détails des routes

#### 🔹 `/status`
Retourne l'état du système Kraken.

**Exemple de réponse :**
```json
{
  "error": [],
  "result": {
    "status": "online",
    "timestamp": "2025-04-02T16:00:00Z"
  }
}
