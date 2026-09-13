package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Tenant struct{ ent.Schema }

func (Tenant) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "tenants"},
	}
}

func (Tenant) Fields() []ent.Field {
	return []ent.Field{
		field.String("id").MaxLen(100).Unique().Immutable(),
		field.String("name").MaxLen(255).NotEmpty(),
		field.String("slug").MaxLen(255).NotEmpty(),
		field.String("domain").MaxLen(255).Optional(),
		field.String("logo").MaxLen(255).Optional(),
		field.Bool("active").Default(true),
		field.String("owner_id").MaxLen(100).Optional(),
		field.String("config").Optional(),
		field.String("settings").Optional(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("deleted_at").Optional().Nillable(),
	}
}

func (Tenant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").Unique().StorageKey("idx_tenants_slug"),
		index.Fields("domain").Unique().StorageKey("idx_tenants_domain"),
		index.Fields("owner_id").StorageKey("idx_tenants_owner"),
		index.Fields("deleted_at").StorageKey("idx_tenants_deleted_at"),
	}
}
