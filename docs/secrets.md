# Secrets con Doppler

El proyecto usa [Doppler](https://doppler.com) como fuente compartida de
secrets (`SUPABASE_URL`, `SUPABASE_ANON_KEY`, `SUPABASE_SERVICE_ROLE_KEY`,
`DATABASE_URL`, `CLOUDFLARE_R2_*`, etc.). Reemplaza pasar `.env` por chat o
email: cada dev se autentica con su propia cuenta y lee los valores actuales
desde Doppler.

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

## Cloudflare R2

El backend guarda los archivos que sube el usuario (el DXF original del alta de
loteo, las fotos y planos de manzana/lote) en un bucket de
[Cloudflare R2](https://developers.cloudflare.com/r2/), que expone la API S3.

Cuatro variables, las cuatro obligatorias y sin default en el repo — mismo
criterio que `SUPABASE_SERVICE_ROLE_KEY`, porque las credenciales dan acceso de
lectura y escritura a todos los archivos del bucket:

| Variable | Qué es |
| --- | --- |
| `CLOUDFLARE_R2_ENDPOINT` | Endpoint S3 de la cuenta: `https://<account-id>.r2.cloudflarestorage.com`. Sin el bucket en la ruta. |
| `CLOUDFLARE_R2_BUCKET_NAME` | Nombre del bucket del entorno (`loteos-files-dev`). |
| `CLOUDFLARE_R2_ACCESS_KEY_ID` | Access Key ID del token de R2. |
| `CLOUDFLARE_R2_SECRET_ACCESS_KEY` | Secret Access Key del token. Solo se muestra al crearlo. |

Si falta cualquiera de las cuatro, `environments.LoadServer` corta el arranque
del backend y nombra todas las que falten de una vez.

### Un bucket por entorno

`loteos-files-dev` para desarrollo, y `loteos-files-stg` / `loteos-files-prd`
cuando existan esos entornos. Buckets separados y no un prefijo dentro de uno
compartido: así una credencial de desarrollo filtrada no alcanza los archivos
de producción, y borrar el bucket de un entorno no toca a los demás.

Cada bucket lleva su propio token, con permiso **Object Read & Write** acotado
a ese bucket. El token de administración (crear buckets, CORS, lifecycles) no
se guarda en Doppler ni se le da al backend: se usa a mano desde el dashboard o
con `wrangler`.

### Crear el bucket

```powershell
pnpm dlx wrangler login
pnpm dlx wrangler r2 bucket create loteos-files-dev --location=enam
```

El token se crea desde el dashboard (**R2 → API → Manage API tokens**), porque
`wrangler` no emite credenciales S3. Al crearlo, cargar los valores en Doppler:

```powershell
doppler secrets set CLOUDFLARE_R2_ACCESS_KEY_ID
doppler secrets set CLOUDFLARE_R2_SECRET_ACCESS_KEY
```

R2 no cobra egress, pero sí storage y operaciones. Cloudflare **no ofrece un
tope de gasto que corte el servicio**: sus
[budget alerts](https://developers.cloudflare.com/billing/manage/budget-alerts/)
solo mandan un email al cruzar un umbral —"informational only. They do not
pause or cap usage"—. Conviene configurarlos igual, pero el control real es
vigilar el uso y el límite de tamaño por archivo del lado de la aplicación.
