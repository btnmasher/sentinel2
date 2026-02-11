export enum EveType {
  Alliance = "alliance",
  Corporation = "corporation",
  Character = "character",
}

export type CharacterAffiliation = EveType.Alliance | EveType.Corporation;
