package internal

import (
	"fmt"
	"github.com/MythicMeta/Mythic_CLI/cmd/config"
	"github.com/MythicMeta/Mythic_CLI/cmd/manager"
	"github.com/MythicMeta/Mythic_CLI/cmd/utils"
	"io"
	"log"
)

// ServiceStart is entrypoint from commands to start containers
func ServiceStart(containers []string) error {
	// first stop all the containers or the ones specified
	_ = manager.GetManager().StopServices(containers, config.GetMythicEnv().GetBool("REBUILD_ON_START"))

	// get all the services on disk and in docker-compose currently
	diskAgents, err := manager.GetManager().GetInstalled3rdPartyServicesOnDisk()
	if err != nil {
		return err
	}
	dockerComposeContainers, err := manager.GetManager().GetAllExistingNonMythicServiceNames()
	if err != nil {
		return err
	}
	intendedMythicServices, err := config.GetIntendedMythicServiceNames()
	if err != nil {
		return err
	}
	currentMythicServices, err := manager.GetManager().GetCurrentMythicServiceNames()
	if err != nil {
		return err
	}
	for _, val := range currentMythicServices {
		if utils.StringInSlice(val, intendedMythicServices) {
		} else {
			_ = manager.GetManager().RemoveServices([]string{val})
		}
	}
	for _, val := range intendedMythicServices {
		AddMythicService(val)
	}
	// if the user didn't explicitly call out starting certain containers, then do all of them
	if len(containers) == 0 {
		containers = append(dockerComposeContainers, intendedMythicServices...)
		// make sure the ports are open that we're going to need
		TestPorts()
	}
	finalContainers := []string{}
	for _, val := range containers { // these are specified containers or all in docker compose
		if !utils.StringInSlice(val, dockerComposeContainers) && !utils.StringInSlice(val, config.MythicPossibleServices) {
			if utils.StringInSlice(val, diskAgents) {
				// the agent mentioned isn't in docker-compose, but is on disk, ask to add
				add := config.AskConfirm(fmt.Sprintf("\n%s isn't in docker-compose, but is on disk. Would you like to add it? ", val))
				if add {
					finalContainers = append(finalContainers, val)
					Add3rdPartyService(val, map[string]interface{}{})
				}
			} else {
				add := config.AskConfirm(fmt.Sprintf("\n%s isn't in docker-compose and is not on disk. Would you like to install it from https://github.com/? ", val))
				if add {
					finalContainers = append(finalContainers, val)
					installServiceByName(val)
				}
			}
		} else {
			finalContainers = append(finalContainers, val)
		}
	}
	for _, service := range finalContainers {
		if utils.StringInSlice(service, config.MythicPossibleServices) {
			AddMythicService(service)
		}
	}
	manager.GetManager().TestPorts()
	err = manager.GetManager().StartServices(finalContainers, config.GetMythicEnv().GetBool("REBUILD_ON_START"))
	err = manager.GetManager().RemoveImages()
	if err != nil {
		fmt.Printf("[-] Failed to remove images\n%v\n", err)
		return err
	}
	updateNginxBlockLists()
	generateCerts()
	TestMythicRabbitmqConnection()
	TestMythicConnection()
	Status(false)
	return nil
}
func ServiceStop(containers []string) error {
	return manager.GetManager().StopServices(containers, config.GetMythicEnv().GetBool("REBUILD_ON_START"))
}
func ServiceBuild(containers []string) error {
	for _, container := range containers {
		if utils.StringInSlice(container, config.MythicPossibleServices) {
			// update the necessary docker compose entries for mythic services
			AddMythicService(container)
		}
	}
	err := manager.GetManager().BuildServices(containers)
	if err != nil {
		return err
	}
	return nil
}
func ServiceRemoveContainers(containers []string) error {
	return manager.GetManager().RemoveContainers(containers)
}

// Docker Save / Load commands

func DockerSave(containers []string) error {
	return manager.GetManager().SaveImages(containers, "saved_images")
}
func DockerLoad() error {
	return manager.GetManager().LoadImages("saved_images")
}
func DockerHealth(containers []string) {
	manager.GetManager().GetHealthCheck(containers)
}

// Build new Docker UI

func DockerBuildReactUI() error {
	if config.GetMythicEnv().GetBool("MYTHIC_REACT_DEBUG") {
		err := manager.GetManager().BuildUI()
		if err != nil {
			log.Fatalf("[-] Failed to build new UI from debug build")
		}
	}
	log.Printf("[-] Not using MYTHIC_REACT_DEBUG to generate new UI, aborting...\n")
	return nil
}

// Docker Volume commands

func VolumesList() {
	manager.GetManager().ListVolumes()
}
func DockerRemoveVolume(volumeName string) error {
	return manager.GetManager().RemoveVolume(volumeName)
}

func DockerCopyIntoVolume(sourceFile io.Reader, destinationFileName string, destinationVolume string) {
	manager.GetManager().CopyIntoVolume(sourceFile, destinationFileName, destinationVolume)
}
func DockerCopyFromVolume(sourceVolumeName string, sourceFileName string, destinationName string) {
	manager.GetManager().CopyFromVolume(sourceVolumeName, sourceFileName, destinationName)
}
