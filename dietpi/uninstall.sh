# xTeVe — DietPi-Software uninstall block
#
# This file is a REFERENCE IMPLEMENTATION showing the code that would be added
# inside Uninstall_Software() in dietpi/dietpi-software for an upstream PR to
# MichaIng/DietPi. It is NOT a standalone script — it is sourced by
# dietpi-software and relies on its helper functions and variables.
#
# Software ID: TBD (assigned by DietPi maintainers)

# --- Uninstall_Software() block ---
if To_Uninstall $software_id # xTeVe
then
	Remove_Service xteve 1
	G_EXEC rm -f /usr/local/bin/xteve
	# Deregister from dietpi-services
	sed -i '/^+xteve$/d' /boot/dietpi/.dietpi-services_include_exclude 2>/dev/null || true
	# User data is intentionally preserved at /mnt/dietpi_userdata/xteve/.
	# Re-installing xTeVe will reuse the existing configuration and EPG database.
	G_WHIP_MSG 'xTeVe has been removed.\n\nUser data at /mnt/dietpi_userdata/xteve/ has been preserved.\nDelete it manually if you no longer need it:\n  rm -rf /mnt/dietpi_userdata/xteve'
fi
