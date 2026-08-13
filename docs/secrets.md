# Secrets con Doppler

El proyecto usa [Doppler](https://doppler.com) como fuente compartida de
secrets (`SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`,
etc.). Reemplaza pasar `.env` por chat o email: cada dev se autentica con su
propia cuenta y lee los valores actuales desde Doppler.

`.env` local sigue existiendo para quien prefiera no usar el CLI (ver
`.env.example`), pero Doppler es la fuente de verdad: si un valor cambia, se
actualiza ahí y cada dev lo vuelve a traer, sin reenviar el archivo.

## Setup inicial (una vez por máquina)

1. Instalar el CLI con [Scoop](https://scoop.sh):

   ```powershell
   scoop bucket add doppler https://github.com/DopplerHQ/scoop-doppler.git
   scoop install doppler
   ```

2. Autenticarse (abre el navegador):

   ```powershell
   doppler login
   ```

3. Pedir acceso al proyecto `loteosapp` en Doppler a quien administre el
   workplace, y luego linkear el repo local al config `dev`:

   ```powershell
   doppler setup --project loteosapp --config dev
   ```

   Este paso guarda la asociación carpeta → proyecto/config en
   `~/.doppler/.doppler.yaml` (fuera del repo); no hay nada que commitear.

## Uso diario

Anteponer `doppler run --` a cualquier comando que necesite los secrets como
variables de entorno:

```powershell
doppler run -- docker compose up --build
```

Para generar un `.env` local puntual (por ejemplo, para herramientas que no
aceptan variables inyectadas):

```powershell
doppler secrets download --no-file --format env > .env
```

## Administrar secrets

```powershell
doppler secrets                 # listar (nombre + valor)
doppler secrets --only-names    # listar solo nombres
doppler secrets set NOMBRE      # crear o actualizar (prompt interactivo)
```

Configs disponibles: `dev`, `stg`, `prd`. Usar `--config <nombre>` para
apuntar a uno distinto de `dev`.
