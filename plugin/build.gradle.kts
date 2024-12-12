import org.jetbrains.kotlin.gradle.dsl.JvmTarget

plugins {
    kotlin("jvm") version "2.1.0"
    kotlin("kapt") version "2.1.0"
    id("com.gradleup.shadow") version "8.3.5"
}

allprojects {
    repositories {
        mavenCentral()
        maven("https://papermc.io/repo/repository/maven-public/")
        maven("https://maven.pkg.github.com/NovusMC/packages") {
            credentials {
                username = project.properties["github_actor"].toString()
                password = project.properties["github_token"].toString()
            }
        }
    }

    group = "eu.novusmc.athena"
    version = "0.1.0"
}

subprojects {

    apply {
        plugin("kotlin")
        plugin("org.jetbrains.kotlin.kapt")
        plugin("com.gradleup.shadow")
    }

    dependencies {
        compileOnly(kotlin("stdlib"))
    }

    kotlin {
        jvmToolchain(21)
    }

    tasks {
        jar {
            enabled = false
        }

        shadowJar {
            destinationDirectory.set(rootProject.layout.buildDirectory.get())
            archiveClassifier.set("")
            archiveVersion.set("")
            archiveBaseName.set("${rootProject.name}-${project.name}")

            minimize()
            exclude("**/*.kotlin_metadata")
            exclude("**/*.kotlin_module")
            exclude("META-INF/maven/**")
        }

        build {
            dependsOn(shadowJar)
        }

        kotlin {
            compilerOptions {
                jvmTarget.set(JvmTarget.JVM_21)
            }
        }
    }
}
