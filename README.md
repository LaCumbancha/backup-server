# Backup Server
TP1 | 75.74 - Sistemas Distribuidos I | 2C2020 | FIUBA

## Requerimientos 

### Funcionales

Se solicita un sistema distribuido que brinde la funcionalidad de backup de las aplicaciones que viven en el cluster de servidores de una empresa. El sistema debe aceptar pedidos por las siguientes operaciones:
* Registrar la dirección de un nuevo nodo del cluster, un puerto, un path y una frecuencia de backups en minutos.
* Consultar los tamaños y fechas de todos los backups realizados para un nodo y path particular.
* Desregistrar un nodo y path para que se dejen de realizar backups.
Los backups deben ser recursivos respecto del path indicado, enviándose al servidor en formato comprimido (tgz) y sólo si dicho archivo posee cambios respecto del último backup realizado. En caso de error en la comunicación o ejecución del backup, el sistema debe reintentar en la próxima oportunidad en que detecte disponibilidad del servidor. 

### No Funcionales

Además del correcto funcionamiento del servidor, deben tenerse en cuenta las siguientes consideraciones:

* Se esperan una gran cantidad aplicaciones que requieren backups que se ejecutan en un conjunto considerable de servidores, por lo que el sistema debe ser fácilmente escalable.
* Los backups se pueden ejecutar 'en caliente', es decir, sin necesidad de interrumpir a las aplicaciones o bloquear los archivos.
* Se debe optimizar la transferencia de información en la red dada la congestión que podrían provocar los volúmenes estimados de backup.
* El servidor de backups debe almacenar un registro total de la ejecución de todos los backups.
* El servidor de backups debe almacenar solamente los últimos 10 archivos de backups de una aplicación y path dados.
* Para ejecutar los pedidos de backup se requiere un cliente liviano que permita invocar las operaciones y recibir confirmación o errores del server.

## Desarrollo

### Ejecución

Para poder levantar el sistema con la configuración inicial, deberá ejecutarse:
```bash
make docker-compose-up
make docker-compose-logs
```

Esta configuración consta de un único nodo para el BackupServer y dos nodos para las aplicaciones dummies. En caso de querer variar estos parámetros, podrá ejecutarse:
```bash
make docker-compose-up BKP_MANAGERS=X ECHO_SERVERS=Y
make docker-compose-logs
```

Dónde se levantarán X servidores e Y aplicaciones. En caso de que se quieran agregar nuevos nodos sin tener que reiniciar el sistema, contamos con el comando:
```bash
make docker-add-echosv
```

Con este se agregará por defecto un único nodo extra, pero pueden especificarse más a través de la variable **NEW_ECHOSVS**. Finalmente, para un correcto cierre del sistema, se deberá utilizar:
```bash
make docker-compose-down
```
### Análisis

Para conectarse directamente con alguno de los containers, podrán utilizarse los siguientes comandos:
```bash
make docker-manager-shell			# Conexión con el BackupServer
make docker-echosv-shell			# Conexión con el EchoServer dummy
```
En caso de que haya más de un container para cada clase, se podrán especificar los parámetros **MANAGER** y **ECHOSV**, indicando a cuál se desea conectar. Por defecto, ambos valores se setearán en 1.
