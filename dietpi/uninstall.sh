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
	G_EXEC rm -Rf /mnt/dietpi_userdata/xteve
fi
