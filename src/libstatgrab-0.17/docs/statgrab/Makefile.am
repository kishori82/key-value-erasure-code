# Makefile for libstatgrab/docs/statgrab
# http://www.i-scream.org/libstatgrab/
# $Id: Makefile.am,v 1.3 2008/05/24 22:30:32 ats Exp $

if MANPAGES
if STATGRAB
man_MANS = statgrab.1 statgrab-make-mrtg-config.1 \
	   statgrab-make-mrtg-index.1
endif
endif

EXTRA_DIST = $(man_MANS)

.xml.1:
	docbook2man -N --symlinks $<

