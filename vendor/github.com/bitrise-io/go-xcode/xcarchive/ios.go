package xcarchive

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-xcode/plistutil"
	"github.com/bitrise-io/go-xcode/profileutil"
)

// IosBaseApplication ...
type IosBaseApplication struct {
	Path                string
	InfoPlist           plistutil.PlistData
	Entitlements        plistutil.PlistData
	ProvisioningProfile profileutil.ProvisioningProfileInfoModel
}

// BundleIdentifier ...
func (app IosBaseApplication) BundleIdentifier() string {
	bundleID, _ := app.InfoPlist.GetString("CFBundleIdentifier")
	return bundleID
}

// NewIosBaseApplication ...
func NewIosBaseApplication(path string) (IosBaseApplication, error) {
	var infoPlist plistutil.PlistData
	{
		infoPlistPath := filepath.Join(path, "Info.plist")
		if exist, err := pathutil.IsPathExists(infoPlistPath); err != nil {
			return IosBaseApplication{}, fmt.Errorf("failed to check if Info.plist exists at: %s, error: %s", infoPlistPath, err)
		} else if !exist {
			return IosBaseApplication{}, fmt.Errorf("Info.plist not exists at: %s", infoPlistPath)
		}
		plist, err := plistutil.NewPlistDataFromFile(infoPlistPath)
		if err != nil {
			return IosBaseApplication{}, err
		}
		infoPlist = plist
	}

	var provisioningProfile profileutil.ProvisioningProfileInfoModel
	{
		provisioningProfilePath := filepath.Join(path, "embedded.mobileprovision")
		if exist, err := pathutil.IsPathExists(provisioningProfilePath); err != nil {
			return IosBaseApplication{}, fmt.Errorf("failed to check if profile exists at: %s, error: %s", provisioningProfilePath, err)
		} else if !exist {
			return IosBaseApplication{}, fmt.Errorf("profile not exists at: %s", provisioningProfilePath)
		}

		profile, err := profileutil.NewProvisioningProfileInfoFromFile(provisioningProfilePath)
		if err != nil {
			return IosBaseApplication{}, err
		}
		provisioningProfile = profile
	}

	executable := executableNameFromInfoPlist(infoPlist)
	entitlements, err := getEntitlements(path, executable)
	if err != nil {
		return IosBaseApplication{}, err
	}

	return IosBaseApplication{
		Path:                path,
		InfoPlist:           infoPlist,
		Entitlements:        entitlements,
		ProvisioningProfile: provisioningProfile,
	}, nil
}

// IosExtension ...
type IosExtension struct {
	IosBaseApplication
}

// NewIosExtension ...
func NewIosExtension(path string) (IosExtension, error) {
	baseApp, err := NewIosBaseApplication(path)
	if err != nil {
		return IosExtension{}, err
	}

	return IosExtension{
		baseApp,
	}, nil
}

// IosWatchApplication ...
type IosWatchApplication struct {
	IosBaseApplication
	Extensions []IosExtension
}

// IosClipApplication ...
type IosClipApplication struct {
	IosBaseApplication
}

// NewIosWatchApplication ...
func NewIosWatchApplication(path string) (IosWatchApplication, error) {
	baseApp, err := NewIosBaseApplication(path)
	if err != nil {
		return IosWatchApplication{}, err
	}

	extensions := []IosExtension{}
	pattern := filepath.Join(pathutil.EscapeGlobPath(path), "PlugIns/*.appex")
	pths, err := filepath.Glob(pattern)
	if err != nil {
		return IosWatchApplication{}, fmt.Errorf("failed to search for watch application's extensions using pattern: %s, error: %s", pattern, err)
	}
	for _, pth := range pths {
		extension, err := NewIosExtension(pth)
		if err != nil {
			return IosWatchApplication{}, err
		}

		extensions = append(extensions, extension)
	}

	return IosWatchApplication{
		IosBaseApplication: baseApp,
		Extensions:         extensions,
	}, nil
}

// NewIosClipApplication ...
func NewIosClipApplication(path string) (IosClipApplication, error) {
	baseApp, err := NewIosBaseApplication(path)
	if err != nil {
		return IosClipApplication{}, err
	}

	return IosClipApplication{
		IosBaseApplication: baseApp,
	}, nil
}

// IosApplication ...
type IosApplication struct {
	IosBaseApplication
	WatchApplication *IosWatchApplication
	ClipApplication  *IosClipApplication
	Extensions       []IosExtension
}

// NewIosApplication ...
func NewIosApplication(path string) (IosApplication, error) {
	baseApp, err := NewIosBaseApplication(path)
	if err != nil {
		return IosApplication{}, err
	}

	var watchApp *IosWatchApplication
	{
		pattern := filepath.Join(pathutil.EscapeGlobPath(path), "Watch/*.app")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			return IosApplication{}, err
		}
		if len(pths) > 0 {
			watchPath := pths[0]
			app, err := NewIosWatchApplication(watchPath)
			if err != nil {
				return IosApplication{}, err
			}
			watchApp = &app
		}
	}

	var clipApp *IosClipApplication
	{
		pattern := filepath.Join(pathutil.EscapeGlobPath(path), "AppClips/*.app")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			return IosApplication{}, err
		}
		if len(pths) > 0 {
			clipPath := pths[0]
			app, err := NewIosClipApplication(clipPath)
			if err != nil {
				return IosApplication{}, err
			}
			clipApp = &app
		}
	}

	extensions := []IosExtension{}
	{
		pattern := filepath.Join(pathutil.EscapeGlobPath(path), "PlugIns/*.appex")
		pths, err := filepath.Glob(pattern)
		if err != nil {
			return IosApplication{}, fmt.Errorf("failed to search for watch application's extensions using pattern: %s, error: %s", pattern, err)
		}
		for _, pth := range pths {
			extension, err := NewIosExtension(pth)
			if err != nil {
				return IosApplication{}, err
			}

			extensions = append(extensions, extension)
		}
	}

	return IosApplication{
		IosBaseApplication: baseApp,
		WatchApplication:   watchApp,
		ClipApplication:    clipApp,
		Extensions:         extensions,
	}, nil
}

// IosArchive ...
type IosArchive struct {
	Path        string
	InfoPlist   plistutil.PlistData
	Application IosApplication
}

// NewIosArchive ...
func NewIosArchive(path string) (IosArchive, error) {
	var infoPlist plistutil.PlistData
	{
		infoPlistPath := filepath.Join(path, "Info.plist")
		if exist, err := pathutil.IsPathExists(infoPlistPath); err != nil {
			return IosArchive{}, fmt.Errorf("failed to check if Info.plist exists at: %s, error: %s", infoPlistPath, err)
		} else if !exist {
			return IosArchive{}, fmt.Errorf("Info.plist not exists at: %s", infoPlistPath)
		}
		plist, err := plistutil.NewPlistDataFromFile(infoPlistPath)
		if err != nil {
			return IosArchive{}, err
		}
		infoPlist = plist
	}

	var application IosApplication
	{
		appPath := ""
		if appRelativePathToProducts, found := applicationFromPlist(infoPlist); found {
			appPath = filepath.Join(path, "Products", appRelativePathToProducts)
		} else {
			var err error
			if appPath, err = applicationFromArchive(path); err != nil {
				return IosArchive{}, err
			}
		}
		if exist, err := pathutil.IsPathExists(appPath); err != nil {
			return IosArchive{}, fmt.Errorf("failed to check if app exists, path: %s, error: %s", appPath, err)
		} else if !exist {
			return IosArchive{}, fmt.Errorf("application not found on path: %s, error: %s", appPath, err)
		}

		app, err := NewIosApplication(appPath)
		if err != nil {
			return IosArchive{}, err
		}
		application = app
	}

	return IosArchive{
		Path:        path,
		InfoPlist:   infoPlist,
		Application: application,
	}, nil
}

func applicationFromPlist(InfoPlist plistutil.PlistData) (string, bool) {
	if properties, found := InfoPlist.GetMapStringInterface("ApplicationProperties"); found {
		return properties.GetString("ApplicationPath")
	}
	return "", false
}

func applicationFromArchive(path string) (string, error) {
	pattern := filepath.Join(pathutil.EscapeGlobPath(path), "Products/Applications/*.app")
	pths, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(pths) == 0 {
		return "", fmt.Errorf("failed to find main app, using pattern: %s", pattern)
	}
	return pths[0], nil
}

// IsXcodeManaged ...
func (archive IosArchive) IsXcodeManaged() bool {
	return archive.Application.ProvisioningProfile.IsXcodeManaged()
}

// SigningIdentity ...
func (archive IosArchive) SigningIdentity() string {
	if properties, found := archive.InfoPlist.GetMapStringInterface("ApplicationProperties"); found {
		identity, _ := properties.GetString("SigningIdentity")
		return identity
	}
	return ""
}

// BundleIDEntitlementsMap ...
func (archive IosArchive) BundleIDEntitlementsMap() map[string]plistutil.PlistData {
	bundleIDEntitlementsMap := map[string]plistutil.PlistData{}

	bundleID := archive.Application.BundleIdentifier()
	bundleIDEntitlementsMap[bundleID] = archive.Application.Entitlements

	for _, plugin := range archive.Application.Extensions {
		bundleID := plugin.BundleIdentifier()
		bundleIDEntitlementsMap[bundleID] = plugin.Entitlements
	}

	if archive.Application.WatchApplication != nil {
		watchApplication := *archive.Application.WatchApplication

		bundleID := watchApplication.BundleIdentifier()
		bundleIDEntitlementsMap[bundleID] = watchApplication.Entitlements

		for _, plugin := range watchApplication.Extensions {
			bundleID := plugin.BundleIdentifier()
			bundleIDEntitlementsMap[bundleID] = plugin.Entitlements
		}
	}

	if archive.Application.ClipApplication != nil {
		clipApplication := *archive.Application.ClipApplication

		bundleID := clipApplication.BundleIdentifier()
		bundleIDEntitlementsMap[bundleID] = clipApplication.Entitlements
	}

	// Add SPM and embedded frameworks that require signing
	frameworkEntitlements := archive.getFrameworkBundleIDEntitlements()
	for bundleID, entitlements := range frameworkEntitlements {
		bundleIDEntitlementsMap[bundleID] = entitlements
	}

	return bundleIDEntitlementsMap
}

// BundleIDProfileInfoMap ...
func (archive IosArchive) BundleIDProfileInfoMap() map[string]profileutil.ProvisioningProfileInfoModel {
	bundleIDProfileMap := map[string]profileutil.ProvisioningProfileInfoModel{}

	bundleID := archive.Application.BundleIdentifier()
	bundleIDProfileMap[bundleID] = archive.Application.ProvisioningProfile

	for _, plugin := range archive.Application.Extensions {
		bundleID := plugin.BundleIdentifier()
		bundleIDProfileMap[bundleID] = plugin.ProvisioningProfile
	}

	if archive.Application.WatchApplication != nil {
		watchApplication := *archive.Application.WatchApplication

		bundleID := watchApplication.BundleIdentifier()
		bundleIDProfileMap[bundleID] = watchApplication.ProvisioningProfile

		for _, plugin := range watchApplication.Extensions {
			bundleID := plugin.BundleIdentifier()
			bundleIDProfileMap[bundleID] = plugin.ProvisioningProfile
		}
	}

	if archive.Application.ClipApplication != nil {
		clipApplication := *archive.Application.ClipApplication

		bundleID := clipApplication.BundleIdentifier()
		bundleIDProfileMap[bundleID] = clipApplication.ProvisioningProfile
	}

	// Add SPM and embedded frameworks that have provisioning profiles
	frameworkProfiles := archive.getFrameworkBundleIDProfiles()
	for bundleID, profile := range frameworkProfiles {
		bundleIDProfileMap[bundleID] = profile
	}

	return bundleIDProfileMap
}

// FindDSYMs ...
func (archive IosArchive) FindDSYMs() ([]string, []string, error) {
	return findDSYMs(archive.Path)
}

// TeamID ...
func (archive IosArchive) TeamID() (string, error) {
	bundleIDProfileInfoMap := archive.BundleIDProfileInfoMap()
	for _, profileInfo := range bundleIDProfileInfoMap {
		return profileInfo.TeamID, nil
	}
	return "", errors.New("team id not found")
}

// getFrameworkBundleIDEntitlements scans the archive for SPM and embedded frameworks that require signing
func (archive IosArchive) getFrameworkBundleIDEntitlements() map[string]plistutil.PlistData {
	frameworkEntitlements := map[string]plistutil.PlistData{}
	
	// Check Frameworks directory in the main application
	frameworksPath := filepath.Join(archive.Application.Path, "Frameworks")
	if exist, err := pathutil.IsPathExists(frameworksPath); err == nil && exist {
		pattern := filepath.Join(pathutil.EscapeGlobPath(frameworksPath), "*.framework")
		frameworks, err := filepath.Glob(pattern)
		if err == nil {
			for _, frameworkPath := range frameworks {
				if bundleID, entitlements := archive.extractFrameworkInfo(frameworkPath); bundleID != "" {
					frameworkEntitlements[bundleID] = entitlements
				}
			}
		}
	}
	
	return frameworkEntitlements
}

// extractFrameworkInfo extracts bundle ID and entitlements from a framework if it has a provisioning profile
func (archive IosArchive) extractFrameworkInfo(frameworkPath string) (string, plistutil.PlistData) {
	// Check if framework has embedded.mobileprovision (indicating it requires signing)
	provisioningProfilePath := filepath.Join(frameworkPath, "embedded.mobileprovision")
	if exist, err := pathutil.IsPathExists(provisioningProfilePath); err != nil || !exist {
		return "", nil
	}
	
	// Read Info.plist to get bundle identifier
	infoPlistPath := filepath.Join(frameworkPath, "Info.plist")
	if exist, err := pathutil.IsPathExists(infoPlistPath); err != nil || !exist {
		return "", nil
	}
	
	infoPlist, err := plistutil.NewPlistDataFromFile(infoPlistPath)
	if err != nil {
		return "", nil
	}
	
	bundleID, ok := infoPlist.GetString("CFBundleIdentifier")
	if !ok || bundleID == "" {
		return "", nil
	}
	
	// Try to get entitlements from the framework executable
	frameworkName := filepath.Base(frameworkPath)
	frameworkName = strings.TrimSuffix(frameworkName, ".framework")
	entitlements, err := getEntitlements(frameworkPath, frameworkName)
	if err != nil {
		// If we can't get entitlements, return empty entitlements but still include the bundle ID
		// This is important for frameworks that need provisioning profiles but don't have entitlements
		entitlements = plistutil.PlistData{}
	}
	
	return bundleID, entitlements
}

// getFrameworkBundleIDProfiles scans the archive for SPM and embedded frameworks that have provisioning profiles
func (archive IosArchive) getFrameworkBundleIDProfiles() map[string]profileutil.ProvisioningProfileInfoModel {
	frameworkProfiles := map[string]profileutil.ProvisioningProfileInfoModel{}
	
	// Check Frameworks directory in the main application
	frameworksPath := filepath.Join(archive.Application.Path, "Frameworks")
	if exist, err := pathutil.IsPathExists(frameworksPath); err == nil && exist {
		pattern := filepath.Join(pathutil.EscapeGlobPath(frameworksPath), "*.framework")
		frameworks, err := filepath.Glob(pattern)
		if err == nil {
			for _, frameworkPath := range frameworks {
				if bundleID, profile := archive.extractFrameworkProfileInfo(frameworkPath); bundleID != "" {
					frameworkProfiles[bundleID] = profile
				}
			}
		}
	}
	
	return frameworkProfiles
}

// extractFrameworkProfileInfo extracts bundle ID and provisioning profile from a framework
func (archive IosArchive) extractFrameworkProfileInfo(frameworkPath string) (string, profileutil.ProvisioningProfileInfoModel) {
	// Check if framework has embedded.mobileprovision
	provisioningProfilePath := filepath.Join(frameworkPath, "embedded.mobileprovision")
	if exist, err := pathutil.IsPathExists(provisioningProfilePath); err != nil || !exist {
		return "", profileutil.ProvisioningProfileInfoModel{}
	}
	
	// Read Info.plist to get bundle identifier
	infoPlistPath := filepath.Join(frameworkPath, "Info.plist")
	if exist, err := pathutil.IsPathExists(infoPlistPath); err != nil || !exist {
		return "", profileutil.ProvisioningProfileInfoModel{}
	}
	
	infoPlist, err := plistutil.NewPlistDataFromFile(infoPlistPath)
	if err != nil {
		return "", profileutil.ProvisioningProfileInfoModel{}
	}
	
	bundleID, ok := infoPlist.GetString("CFBundleIdentifier")
	if !ok || bundleID == "" {
		return "", profileutil.ProvisioningProfileInfoModel{}
	}
	
	// Read the provisioning profile
	profile, err := profileutil.NewProvisioningProfileInfoFromFile(provisioningProfilePath)
	if err != nil {
		return "", profileutil.ProvisioningProfileInfoModel{}
	}
	
	return bundleID, profile
}
