package network

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/vishvananda/netlink"
)

func ensureBridge(name, gatewayCIDR string) error {
	addr, err := netlink.ParseAddr(gatewayCIDR)
	if err != nil {
		return fmt.Errorf("Error Parsing Address:%w", err)
	}
	link, err := netlink.LinkByName(name)
	var notFound netlink.LinkNotFoundError
	if err != nil {
		if !errors.As(err, &notFound) {
			return fmt.Errorf("Error creating bridage %w", err)
		}
		la := netlink.NewLinkAttrs()
		la.Name = name
		mybridge := &netlink.Bridge{
			LinkAttrs: la,
		}
		link = mybridge
		err = netlink.LinkAdd(link)
		if err != nil {
			return fmt.Errorf("could not add %s: %w", link.Attrs().Name, err)
		}
	}

	err = netlink.AddrReplace(link, addr)
	if err != nil {
		return fmt.Errorf("Error Replace address:%w", err)
	}
	err = netlink.LinkSetUp(link)
	if err != nil {
		return fmt.Errorf("Error Setup bridge:%w", err)
	}
	log.Printf("Successfully setup bridge !")
	return nil
}

func ensureTap(name string, bridge string) error {
	link, err := netlink.LinkByName(name)
	var notFound netlink.LinkNotFoundError
	if err != nil {
		if !errors.As(err, &notFound) {
			return fmt.Errorf("Error creating tap %w", err)
		}
		//ip tuntap add dev tap0 mode tap user mingdaxie
		la := netlink.NewLinkAttrs()
		la.Name = name
		tap := netlink.Tuntap{
			LinkAttrs: la,
			Mode:      netlink.TUNTAP_MODE_TAP,
			Owner:     uint32(os.Getuid()),
		}
		link = &tap
		err := netlink.LinkAdd(link)
		if err != nil {
			return fmt.Errorf("Error Adding link :%w", err)
		}
	}
	br0, err := netlink.LinkByName(bridge)
	if err != nil {
		return fmt.Errorf("Can not find bridge:%w", err)
	}
	//ip link set tap0 master br0
	err = netlink.LinkSetMaster(link, br0)
	if err != nil {
		return fmt.Errorf("Error setting bridge %s to master :%w", br0.Attrs().Name, err)
	}
	// ip link set tap0 up
	err = netlink.LinkSetUp(link)
	if err != nil {
		return fmt.Errorf("Error Setup tap:%w", err)
	}
	log.Printf("Setup master complete !")
	return nil
}

func Setup(bridge, gatewayCIDR string, n int) error {
	if err := ensureBridge(bridge, gatewayCIDR); err != nil {
		return fmt.Errorf("Error setting up Bridge:%w", err)
	}
	for i := range n {
		err := ensureTap(fmt.Sprintf("tap%d", i), bridge)
		if err != nil {
			return fmt.Errorf("Failed to create tap at %d : %w", i, err)
		}
	}
	return nil
}
