package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type ExtensionType int32
type HeroId string

type HeroSkill struct {
	Number       int    // skin1.png -> SkillNumber = 1
	AbsolutePath string // absolute path to file
}

type Skin struct {
	Number        int    // use to refactor directory and generate hero spine
	AbsolutePath  string // path to file, absolute path, e.g. "C:/data/hero_XXXX_YYYY/web_idle/Skin1/YYYY.json"
	FileName      string // just file name, e.g. "YYYY"
	FileExtension string // e.g. "json"
}

const (
	PNG ExtensionType = iota
	JSON
	ATLAS
	UNDEFINED
)

func getExtensionType(path string) ExtensionType {
	switch {
	case strings.HasSuffix(path, ".json"):
		return JSON
	case strings.HasSuffix(path, ".atlas"):
		return ATLAS
	case strings.HasSuffix(path, ".png"):
		return PNG
	default:
		return UNDEFINED
	}
}

// getAllPath dfs to get all path
func getAllPath(root string, lstPath *[]string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		log.Fatal(err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			getAllPath(root+"\\"+entry.Name(), lstPath)
		} else {
			*lstPath = append(*lstPath, string(root+"\\"+entry.Name()))
		}
	}
}

// skillFromPath take a path and extract the hero's id and hero's skill info
// the path consider valid if it has the below form
// "..../icon_skill_XXXX_Y.png"
// XXXX: hero's id
// Y: skin number
func skillFromPath(path string) (HeroId, *HeroSkill, error) {
	components := strings.Split(path, "\\")
	extension := getExtensionType(path)
	if len(components) < 2 || extension != PNG {
		return "", nil, errors.New("undefined path")
	}
	// just take last component to work
	lastComponent := components[len(components)-1]
	if !strings.HasPrefix(lastComponent, "icon_skill_") || strings.Count(lastComponent, "_") != 3 {
		return "", nil, errors.New("wrong format")
	}

	lastComponent = lastComponent[11 : len(lastComponent)-4]
	pos := 0
	for pos < len(lastComponent) && lastComponent[pos] != '_' {
		pos++
	}
	if pos == 0 {
		return "", nil, errors.New("wrong format")
	}

	id := lastComponent[:pos]
	skillNumber, err := strconv.Atoi(lastComponent[pos+1:])
	if err != nil {
		return "", nil, err
	}

	start := 0
	for i := 0; i < len(components)-2; i++ {
		start += len(components[i])
	}
	start += len(components) - 3

	return HeroId(id), &HeroSkill{Number: skillNumber, AbsolutePath: path}, nil
}

// skinFromPath take an path and extract the hero's id and Skin
// Path with form below consider valid
// "..../hero_XXXX_YYYY/.../SkinZ/YYYY.*"
// XXXX: hero's id
// YYYY: hero's name
// Z: skin number
func skinFromPath(path string) (HeroId, *Skin, error) {
	components := strings.Split(path, "\\")
	extension := getExtensionType(path)
	if len(components) < 3 || extension == UNDEFINED {
		return "", nil, errors.New("undefined path")
	}

	heroComponent, skinComponent := "", ""

	for _, component := range components {
		if strings.HasPrefix(component, "hero_") && strings.Count(component, "_") == 2 {
			heroComponent = component
		}
		if strings.HasPrefix(component, "Skin") && len(component) > 4 {
			skinComponent = component
		}
	}

	if heroComponent == "" || skinComponent == "" {
		return "", nil, errors.New("undefined path")
	}

	i := 0
	for i < len(heroComponent) && heroComponent[i] != '_' {
		i++
	}
	j := i + 1
	for j < len(heroComponent) && heroComponent[j] != '_' {
		j++
	}
	if i >= len(heroComponent) || j >= len(heroComponent) || heroComponent[i] != '_' || heroComponent[j] != '_' {
		return "", nil, errors.New("undefined path")
	}
	heroId := heroComponent[i+1 : j]

	skinNumber, err := strconv.Atoi(skinComponent[4:])
	if err != nil {
		return "", nil, errors.New("wrong format: can't convert skin number, folder should be in form 'SkinX'")
	}
	lastComponent := components[len(components)-1]
	fileName := lastComponent[:len(lastComponent)-int(extension)-4]
	fileExtension := lastComponent[len(fileName)+1:]

	return HeroId(heroId), &Skin{Number: skinNumber, AbsolutePath: path, FileName: fileName, FileExtension: fileExtension}, nil
}

// get skill name from csv file
// return map(heroId_skillNumber => skillName)
func getSkillId(csvPath string) (map[string]string, error) {
	data, err := os.ReadFile(csvPath)
	if err != nil {
		return nil, err
	}
	rows := strings.Split(string(data), "\n")
	ret := make(map[string]string)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		columns := strings.Split(row, ",")
		if len(columns) >= 2 {
			ret[columns[0]] = columns[1]
		}
	}
	return ret, nil
}

func Copy(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func main() {
	args := os.Args
	if len(args) < 4 {
		msg := "Please provide path to skin and skill folder\n" + "You must run command like: '%s \"[hero's skill path]\" \"[hero's skin path]\" \"[csv's skill path]\"'\n" + "E.g. '%s \"C:\\folder\\skin\\hero\" \"C:\\folder\\skill\\hero\" \"C:\\folder\\skill\\skillInfo.csv\"'"
		log.Fatalf(msg, args[0], args[0])
	}
	//-------------------------------------------------------------------------
	log.Println("Refactoring directory...")

	skillPath, skinPath, skillCsvPath := args[1], args[2], args[3]
	lstSkillPath, lstSkinPath := make([]string, 0), make([]string, 0)
	unknownSkillPath, unknownSkinPath := make([]string, 0), make([]string, 0)

	skillInfo, skinInfo := make(map[HeroId][]*HeroSkill), make(map[HeroId][]*Skin)

	getAllPath(skillPath, &lstSkillPath)
	for i := 0; i < len(lstSkillPath); i++ {
		heroId, skill, err := skillFromPath(lstSkillPath[i])
		if err != nil {
			unknownSkillPath = append(unknownSkillPath, lstSkillPath[i]+" ("+err.Error()+")")
		} else {
			skillInfo[heroId] = append(skillInfo[heroId], skill)
		}
	}

	getAllPath(skinPath, &lstSkinPath)
	for i := 0; i < len(lstSkinPath); i++ {
		heroId, skin, err := skinFromPath(lstSkinPath[i])
		if err != nil {
			unknownSkinPath = append(unknownSkinPath, lstSkinPath[i]+" ("+err.Error()+")")
		} else {
			skinInfo[heroId] = append(skinInfo[heroId], skin)
		}
	}

	// write result to folder tree
	// hero
	// -- hero_id
	// ---- skin1
	// ------ oceanee.json
	// ---- ...
	// ---- skill
	// ------ icon1.png
	// ------ icon2.png
	// ------ ...
	// -- ...

	// remove old data
	os.RemoveAll("hero")

	for heroId, skills := range skillInfo {
		path := "hero/" + string(heroId) + "/"
		for _, skill := range skills {
			if skill == nil {
				log.Fatal("unexpected behavior")
			}
			os.MkdirAll(path+"skill", os.ModePerm)
			if err := Copy(skill.AbsolutePath, path+"skill"+"/icon"+strconv.FormatInt(int64(skill.Number), 10)+".png"); err != nil {
				log.Println(err)
			}
		}
	}

	for heroId, skins := range skinInfo {

		path := "hero/" + string(heroId) + "/"
		for _, skin := range skins {
			if skin == nil {
				log.Fatal("unexpected behavior")
			}
			skinName := "skin" + strconv.FormatInt(int64(skin.Number), 10)
			os.MkdirAll(path+skinName, os.ModePerm)
			fileName := skin.FileName + "." + skin.FileExtension
			if err := Copy(skin.AbsolutePath, path+skinName+"/"+fileName); err != nil {
				log.Println(err)
			}
		}
	}

	// unknown path should be written to trace
	os.WriteFile("DeletedSkillPath.txt", []byte(strings.Join(unknownSkillPath, "\n")), os.ModePerm)
	os.WriteFile("DeletedSkinPath.txt", []byte(strings.Join(unknownSkinPath, "\n")), os.ModePerm)

	log.Println("Refactoring directory... OK!")
	//-------------------------------------------------------------------------
	log.Println("Generating hero spine...")

	heroSpine := make(map[string]string)
	for heroId, skins := range skinInfo {
		for _, skin := range skins {
			if skin != nil && skin.FileExtension == "json" {
				heroSpine[string(heroId)+"_"+strconv.FormatInt(int64(skin.Number), 10)] = skin.FileName
			}
		}
	}

	jsonData, err := json.Marshal(heroSpine)
	if err != nil {
		log.Fatal(err)
	}
	os.WriteFile("heroSpine.json", jsonData, 0777)

	log.Println("Generating hero spine... OK!")
	//-------------------------------------------------------------------------
	log.Println("Generating hero skill and check wrong synchronized data...")

	heroSkill := make(map[HeroId][]*string)

	skillName, err := getSkillId(skillCsvPath)
	if err != nil {
		log.Fatal(err)
	}

	wrongSynchronizedData := make([]string, 0)

	for heroId, skills := range skillInfo {
		hasSkillIcon := make(map[int]bool)
		for _, skill := range skills {
			hasSkillIcon[skill.Number] = true
		}
		for i := 1; i <= 4; i++ {
			sValue := skillName[string(heroId)+"_"+strconv.FormatInt(int64(i), 10)]
			if hasSkillIcon[i] {
				// have skill icon but doesn't have skill name
				if sValue == "" {
					wrongSynchronizedData = append(wrongSynchronizedData, string(heroId))
				}
				heroSkill[heroId] = append(heroSkill[heroId], &sValue)
			} else {
				// have skill name but doesn't have skill icon
				if sValue != "" {
					wrongSynchronizedData = append(wrongSynchronizedData, string(heroId))
				}
				heroSkill[heroId] = append(heroSkill[heroId], nil)
			}
		}
	}

	jsonData, err = json.Marshal(heroSkill)
	if err != nil {
		log.Fatal(err)
	}
	os.WriteFile("heroSkill.json", jsonData, 0777)
	os.WriteFile("WrongSynchronizedData.txt", []byte(strings.Join(wrongSynchronizedData, "\n")), os.ModePerm)

	log.Println("Generating hero skill and check wrong synchronized data... OK!")
}

func init() {

}
